package dsl

import (
	"bytes"
	"errors"
	"github.com/serverlessworkflow/sdk-go/v2/model"
	"go.temporal.io/sdk/workflow"
	"text/template"
	"time"
)

const TaskQueue = "ServerlessWorkflowTaskQueue"

// global for now, may need to embed into context
var sw *model.Workflow

// ServerlessWorkflow workflow definition with a lot of simplifications
// TODO store the *model.Workflow in context as it will be needed all the way through
// TODO do not support EventState for now everything should be Transition
func ServerlessWorkflow(ctx workflow.Context, sworkflow *model.Workflow, wfInput string) (string, error) {

	logger := workflow.GetLogger(ctx)
	logger.Info("Serverless Workflow started.")

	// embed the sw into context
	sw = sworkflow

	var start model.State
	for _, st := range sw.States {
		if st.GetName() == sw.Start.StateName {
			start = st
			break
		}
	}

	// POC, the ws does allow this
	if start == nil {
		return "", errors.New("start is not a valid state")
	}

	result, _ := startSWExec(ctx, start, wfInput)

	logger.Debug("workflow", sw)

	logger.Info("Serverless Workflow completed.")

	return result, nil
}

func startSWExec(ctx workflow.Context, start model.State, input string) (string, error) {
	// TODO EventState not supported yet
	// if the first state is an EventState then we just simply fire the event from the input
	var (
		output string
		err    error
		next   = start
	)

	for {
		if output, next, err = execState(ctx, next, input); err != nil {
			return "", err
		}

		if next == nil {
			break
		}
	}

	return output, nil
}

func getNextState(tzn model.Transition) model.State {
	for _, s := range sw.States {
		if s.GetName() == tzn.NextState {
			return s
		}
	}
	return nil
}

func execState(ctx workflow.Context, state model.State, input string) (string, model.State, error) {
	logger := workflow.GetLogger(ctx)

	var (
		output string
		next   model.State
		err    error
	)

	switch t := state.(type) {
	case *model.OperationState:
		output, next, err = execOperation(ctx, t, input)
	case *model.DataBasedSwitchState:
		output, next, err = execSwitch(ctx, t, input)
	default:
		logger.Error("state not supported")
	}

	return output, next, err
}

func execOperation(ctx workflow.Context, s *model.OperationState, data string) (string, model.State, error) {
	// TODO check if a.FunctionRef.RefName is a registered func by reflection on Activities
	// TODO execute in parallel indicated by s.ActionMode
	var res string
	for _, a := range s.Actions {
		// ignore s.Timeouts;
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			TaskQueue:           TaskQueue,
		})
		we := workflow.ExecuteActivity(ctx, a.FunctionRef.RefName, data)
		err := we.Get(ctx, &res)
		if err != nil {
			return "", nil, err
		}
	}

	var next model.State
	if s.GetTransition() != nil {
		next = getNextState(*s.GetTransition())
	}

	return res, next, nil
}

func execSwitch(ctx workflow.Context, s *model.DataBasedSwitchState, input string) (string, model.State, error) {
	customer, err := toCustomer(input)
	if err != nil {
		return "", nil, err
	}

	// branch out on the first matching condition
	var condition model.DataCondition
	for _, c := range s.DataConditions {
		if ok, err := evalCustomerCondition(customer, c.GetCondition()); err != nil {
			return "", nil, err
		} else if ok {
			condition = c
			break
		}
	}

	if condition == nil {
		return "", nil, nil
	}

	switch c := condition.(type) {
	case *model.TransitionDataCondition:
		return input, getNextState(c.Transition), nil
	case *model.EndDataCondition:
		return input, nil, nil
	default:
		return "", nil, errors.New("unsupported condition")
	}
}

func evalCustomerCondition(customer *Customer, condition string) (bool, error) {
	tmpl := template.Must(template.New("condition").Parse(condition))
	var buf bytes.Buffer

	err := tmpl.Execute(&buf, customer)
	if err != nil {
		return false, err
	}

	return buf.String() == "true", nil
}
