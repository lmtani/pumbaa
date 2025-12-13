package cromwell

import "github.com/lmtani/pumbaa/internal/domain/workflow"

// mapMetadataToWorkflow converts a metadata response to a domain Workflow.
func (c *Client) mapMetadataToWorkflow(m *metadataResponse) *workflow.Workflow {
	wf := &workflow.Workflow{
		ID:          m.ID,
		Name:        m.WorkflowName,
		Status:      workflow.Status(m.Status),
		Start:       m.Start,
		End:         m.End,
		SubmittedAt: m.Submission,
		Labels:      m.Labels,
		Inputs:      m.Inputs,
		Outputs:     m.Outputs,
		Calls:       make(map[string][]workflow.Call),
		Failures:    make([]workflow.Failure, 0),
	}

	// Map calls
	for callName, calls := range m.Calls {
		wf.Calls[callName] = make([]workflow.Call, 0, len(calls))
		for _, call := range calls {
			wf.Calls[callName] = append(wf.Calls[callName], workflow.Call{
				Name:              callName,
				Status:            workflow.Status(call.ExecutionStatus),
				Start:             call.Start,
				End:               call.End,
				Attempt:           call.Attempt,
				ShardIndex:        call.ShardIndex,
				Backend:           call.Backend,
				ReturnCode:        call.ReturnCode,
				Stdout:            call.Stdout,
				Stderr:            call.Stderr,
				CommandLine:       call.CommandLine,
				Inputs:            call.Inputs,
				Outputs:           call.Outputs,
				RuntimeAttributes: call.RuntimeAttributes,
				Failures:          mapFailures(call.Failures),
				SubWorkflowID:     call.SubWorkflowID,
			})
		}
	}

	// Map failures
	wf.Failures = mapFailures(m.Failures)

	return wf
}

// mapFailures converts failure metadata to domain failures.
func mapFailures(failures []failureMetadata) []workflow.Failure {
	result := make([]workflow.Failure, 0, len(failures))
	for _, f := range failures {
		result = append(result, workflow.Failure{
			Message:  f.Message,
			CausedBy: mapFailures(f.CausedBy),
		})
	}
	return result
}
