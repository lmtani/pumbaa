package telemetry

// NoOpService is a telemetry service that does nothing
type NoOpService struct{}

func NewNoOpService() *NoOpService {
	return &NoOpService{}
}

func (s *NoOpService) Track(event Event) {}

func (s *NoOpService) TrackCommand(ctx CommandContext, err error) {}

func (s *NoOpService) CaptureError(operation string, err error) {}

func (s *NoOpService) AddBreadcrumb(category, message string) {}

func (s *NoOpService) Close() {}
