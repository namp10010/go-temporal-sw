package dsl

import (
	"context"
	"encoding/json"
	"fmt"

	"go.temporal.io/sdk/activity"
)

type OnboardingActivities struct {
}

func (a *OnboardingActivities) CheckCustomerInfo(ctx context.Context, input string) (string, error) {
	name := activity.GetInfo(ctx).ActivityType.Name
	fmt.Printf("Run %s with input %s \n", name, string(input))
	return "Result_" + name, nil
}

func (a *OnboardingActivities) UpdateApplicationInfo(ctx context.Context, input string) (string, error) {
	name := activity.GetInfo(ctx).ActivityType.Name
	fmt.Printf("Run %s with input %s \n", name, string(input))
	return "Result_" + name, nil
}

func (a *OnboardingActivities) ApproveApplication(ctx context.Context, input string) (string, error) {
	fmt.Printf("Run %s with input %s \n", activity.GetInfo(ctx).ActivityType.Name, input)
	return `{"ApproveApplication":"true"}`, nil
}

func (a *OnboardingActivities) RejectApplication(ctx context.Context, input string) (string, error) {
	fmt.Printf("Run %s with input %s \n", activity.GetInfo(ctx).ActivityType.Name, input)
	return `{"RejectApplication":"true"}`, nil
}

type Customer struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Age       int    `json:"age"`
	Request   string `json:"request"`
}

func toCustomer(input string) (*Customer, error) {
	wrappedCustomer := struct {
		Customer *Customer `json:"customer"`
	}{}
	err := json.Unmarshal([]byte(input), &wrappedCustomer)
	return wrappedCustomer.Customer, err
}
