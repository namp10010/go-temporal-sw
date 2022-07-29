package main

import (
	"context"
	"dsl"
	"encoding/json"
	"github.com/serverlessworkflow/sdk-go/v2/parser"
	"github.com/serverlessworkflow/sdk-go/v2/validator"
	"go.temporal.io/sdk/client"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type starter struct {
	c client.Client
}

// Location of static content
// use "./starter/html" when running from root directory
// use "./html" when running from current directory
var static = "./starter/html"

func main() {
	// Static live editor
	if _, err := os.Stat(static); os.IsNotExist(err) {
		log.Fatal(static + " does not exist")
	}
	fs := http.FileServer(http.Dir(static))
	http.Handle("/", fs)

	log.Print("editor listening on :3000...")
	go func() {
		log.Fatal(http.ListenAndServe(":3000", nil))
	}()

	// The client is a heavyweight object that should be created once per process.
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	s := starter{c: c}
	http.HandleFunc("/runworkflow", s.runWorkflow)

	log.Print("starter listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s starter) runWorkflow(writer http.ResponseWriter, request *http.Request) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "sw-localtest",
		TaskQueue: "ServerlessWorkflowTaskQueue",
	}

	defer request.Body.Close()

	reqBody := struct {
		Data string `json:"workflowdata"`
		DSL  string `json:"workflowdsl"`
	}{}

	body, _ := ioutil.ReadAll(request.Body)
	if err := json.Unmarshal(body, &reqBody); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}

	workflow, err := parser.FromJSONSource([]byte(reqBody.DSL))
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}

	if err = validator.GetValidator().Struct(workflow); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}

	// TODO validate actions/activities to registered ones

	we, err := s.c.ExecuteWorkflow(context.Background(), workflowOptions, dsl.ServerlessWorkflow, workflow, reqBody.Data)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// Synchronously wait for the workflow completion.
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable get workflow result", err)
	}
	log.Println("Workflow result:", string(result))

	// Write response
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(result))
}
