package tools

type dataHandler func(*DataPacket) error

// Config manages the configuration of the client and server objects
type Config struct {
	// When initiating a server, Host is the IP to listen on
	// When initiating a client, Host is the target nsca host
	Host string
	// When initializing a server, Port is the port to listen on
	// When initializing a client, Port is the port of the target nsca server
	Port uint16
	// Encryption method to use based on the standard NSCA encryptions list
	EncryptionMethod int
	// Password is used to encrypt and decrypt the messages if EncryptionMethod is
	// not set to 0.
	Password string
	// Max size of each fields
	MaxHostnameSize     uint16
	MaxDescriptionSize  uint16
	MaxPluginOutputSize uint16
	// PacketHandler is the function that will handle the DataPacket in the
	// HandleClient function. You define what you want to do with the DataPacket
	// once decrypted and transformed to a DataPacket struct.
	// This function should follow this: func(*dataPacket) error
	PacketHandler dataHandler
}

// NewConfig initiates a Config object (with default values if zero values given)
func NewConfig(host string, port uint16, encryption int, password string, handler dataHandler) *Config {
	cfg := Config{
		EncryptionMethod:    encryption,
		Password:            password,
		MaxHostnameSize:     64,
		MaxDescriptionSize:  128,
		MaxPluginOutputSize: 4096,
		PacketHandler:       handler,
	}
	if cfg.Host = "localhost"; host != "" {
		cfg.Host = host
	}
	if cfg.Port = 5667; port != 0 {
		cfg.Port = port
	}
	return &cfg
}
