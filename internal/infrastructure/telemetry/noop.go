package telemetry

// NoOpService is a telemetry service that does nothing
type NoOpService struct{}

func NewNoOpService() *NoOpService {
	return &NoOpService{}
}

func (s *NoOpService) Track(event Event) {}

func (s *NoOpService) Close() {}
