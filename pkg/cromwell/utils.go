package cromwell

import (
	"context"
	"log"

	"google.golang.org/api/idtoken"
)

func submitPrepare(r SubmitRequest) map[string]string {
	fileParams := map[string]string{
		"workflowSource": r.WorkflowSource,
		"workflowInputs": r.WorkflowInputs,
	}
	if r.WorkflowDependencies != "" {
		fileParams["workflowDependencies"] = r.WorkflowDependencies
	}
	if r.WorkflowOptions != "" {
		fileParams["workflowOptions"] = r.WorkflowOptions
	}
	return fileParams
}

func getGoogleIapToken(aud string) string {
	ctx := context.Background()
	ts, err := idtoken.NewTokenSource(ctx, aud)
	if err != nil {
		log.Fatal(err)
	}
	token, err := ts.Token()
	if err != nil {
		log.Fatal(err)
	}
	return token.AccessToken
}
