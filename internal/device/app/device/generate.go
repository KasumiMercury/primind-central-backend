package device

//go:generate mockgen -destination=mock_auth_client.go -package=device github.com/KasumiMercury/primind-central-backend/internal/device/infra/authclient AuthClient
//go:generate mockgen -destination=mock_device_repository.go -package=device github.com/KasumiMercury/primind-central-backend/internal/device/domain/device DeviceRepository
