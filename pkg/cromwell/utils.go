package cromwell

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

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

func errorHandler(r *http.Response) error {
	var er = ErrorResponse{
		HTTPStatus: r.Status,
	}
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		log.Println("No json body in response")
	}
	return errors.New(fmt.Sprintf("Submission failed. The server returned %#v", er))
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
