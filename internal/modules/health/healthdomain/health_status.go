package healthdomain

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

func ToHealthStatus(healthy bool) HealthStatus {
	if healthy {
		return HealthStatusHealthy
	}
	return HealthStatusUnhealthy
}
