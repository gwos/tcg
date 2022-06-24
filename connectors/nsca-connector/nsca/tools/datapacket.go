package tools

import "C"
import (
	"fmt"
	"hash/crc32"
	"time"
	"unsafe"
)

// StateUnknown is the state understood by NSCA
const (
	StateUnknown = iota
)

// Encrypt* are the encryptions supported by the standard NSCA configuration
const (
	EncryptNone        = iota // no encryption
	EncryptXOR                // Simple XOR  (No security, just obfuscation, but very fast)
	EncryptDES                // DES
	Encrypt3DES               // 3DES or Triple DES
	EncryptCAST128            // CAST-128
	EncryptCAST256            // CAST-256
	EncryptXTEA               // xTEA
	Encrypt3WAY               // 3-WAY
	EncryptBLOWFISH           // SKIPJACK
	EncryptTWOFISH            // TWOFISH
	EncryptLOKI97             // LOKI97
	EncryptRC2                // RC2
	EncryptARCFOUR            // RC4
	EncryptRC6                // RC6 - Unsupported in standard NSCA
	EncryptRIJNDAEL128        // AES-128
	EncryptRIJNDAEL192        // AES-192
	EncryptRIJNDAEL256        // AES-256
	EncryptMARS               // MARS - Unsupported in standard NSCA
	EncryptPANAMA             // PANAMA - Unsupported in standard NSCA
	EncryptWAKE               // WAKE
	EncryptSERPENT            // SERPENT
	EncryptIDEA               // IDEA - Unsupported in standard NSCA
	EncryptENIGMA             // ENIGMA (Unix crypt)
	EncryptGOST               // GOST
	EncryptSAFER64            // SAFER-sk64
	EncryptSAFER128           // SAFER-sk128
	EncryptSAFERPLUS          // SAFER+
)

// DataPacket stores the data received for the client-server communication
type DataPacket struct {
	Version      int16
	Crc          uint32
	Timestamp    uint32
	State        int16
	HostName     string
	Service      string
	PluginOutput string
	Ipkt         *InitPacket
	Password     []byte
	Encryption   int
}

// NewDataPacket initializes a new blank data packet
func NewDataPacket(encryption int, password []byte, ipkt *InitPacket) *DataPacket {
	packet := DataPacket{
		Version:      3,
		Crc:          0,
		Timestamp:    uint32(time.Now().Unix()),
		State:        StateUnknown,
		HostName:     "",
		Service:      "",
		PluginOutput: "",
		Ipkt:         ipkt,
		Password:     password,
		Encryption:   encryption,
	}
	return &packet
}

// Decrypt decrypts a buffer
func (p *DataPacket) Decrypt(buffer []byte) error {
	var (
		algo string
		err  error
	)

	switch p.Encryption {
	case EncryptNone: // Just don't do anything
	case EncryptXOR:
		p.xor(buffer)
	case EncryptIDEA: // Unsupported in standard NSCA
		err = fmt.Errorf("unimplemented encryption algorithm")
	case EncryptRC6: // Unsupported in standard NSCA
		err = fmt.Errorf("unimplemented encryption algorithm")
	case EncryptMARS: // Unsupported in standard NSCA
		err = fmt.Errorf("unimplemented encryption algorithm")
	case EncryptPANAMA: // Unsupported in standard NSCA
		err = fmt.Errorf("unimplemented encryption algorithm")
	default:
		algo = p.setAlgo()
		if algo == "Unknown" {
			err = fmt.Errorf("%d is an unrecognized encryption integer", p.Encryption)
		}
	}
	if err != nil {
		return err
	}
	if algo != "" {
		err = MCrypt(algo, buffer, p.Password, p.Ipkt.Iv, true)
	}
	return err
}

// Performs a XOR operation on a buffer using the initialization vector and the
// password.
func (p *DataPacket) xor(buffer []byte) {
	bufferSize := len(buffer)
	ivSize := len(p.Ipkt.Iv)
	pwdSize := len(p.Password)
	// Rotating over the initialization vector of the connection and the password
	for y := 0; y < bufferSize; y++ {
		buffer[y] ^= p.Ipkt.Iv[y%ivSize] ^ p.Password[y%pwdSize]
	}
}

// setAlgo translates the encryption to an mcrypt-understandable from algorithm name
func (p *DataPacket) setAlgo() string {
	var algo string
	switch p.Encryption {
	case EncryptDES:
		algo = "des"
	case Encrypt3DES:
		algo = "tripledes"
	case EncryptCAST128:
		algo = "cast-128"
	case EncryptCAST256:
		algo = "cast-256"
	case EncryptXTEA:
		algo = "xtea"
	case Encrypt3WAY:
		algo = "threeway"
	case EncryptBLOWFISH:
		algo = "blowfish"
	case EncryptTWOFISH:
		algo = "twofish"
	case EncryptLOKI97:
		algo = "loki97"
	case EncryptRC2:
		algo = "rc2"
	case EncryptARCFOUR:
		algo = "arcfour"
	case EncryptRIJNDAEL128:
		algo = "rijndael-128"
	case EncryptRIJNDAEL192:
		algo = "rijndael-192"
	case EncryptRIJNDAEL256:
		algo = "rijndael-256"
	case EncryptWAKE:
		algo = "wake"
	case EncryptSERPENT:
		algo = "serpent"
	case EncryptENIGMA:
		algo = "enigma"
	case EncryptGOST:
		algo = "gost"
	case EncryptSAFER64:
		algo = "safer-sk64"
	case EncryptSAFER128:
		algo = "safer-sk128"
	case EncryptSAFERPLUS:
		algo = "saferplus"
	default:
		algo = "Unknown"
	}
	return algo
}

// MCrypt uses libmcrypt to decrypt/encrypt the data received from the send_nsca
// client. When I have some more time I'll dig to find out why I'm not able to
// decrypt directly using the NewCFBDecrypter
// To decrypt, set the decrypt parameter to true, else it will encrypt.
func MCrypt(algo string, blocks, key, iv []byte, decrypt bool) error {
	algorithm := C.CString(algo)
	defer C.free(unsafe.Pointer(algorithm))

	mode := C.CString("cfb")
	defer C.free(unsafe.Pointer(mode))

	td := C.mcrypt_module_open(algorithm, nil, mode, nil)
	defer C.mcrypt_module_close(td)

	if uintptr(unsafe.Pointer(td)) == C.MCRYPT_FAILED {
		return fmt.Errorf("mcrypt module open failed")
	}

	keySize := C.mcrypt_enc_get_key_size(td)
	ivSize := C.mcrypt_enc_get_iv_size(td)
	realKey := make([]byte, keySize)
	if len(key) > int(keySize) {
		copy(realKey, key[:keySize])
	} else {
		copy(realKey, key)
	}
	realIv := make([]byte, ivSize)
	if len(iv) > int(ivSize) {
		copy(realIv, iv[:ivSize])
	} else {
		copy(realIv, iv)
	}

	rv := C.mcrypt_generic_init(td, unsafe.Pointer(&realKey[0]), keySize, unsafe.Pointer(&realIv[0]))
	defer C.mcrypt_generic_deinit(td)

	if rv < 0 {
		return fmt.Errorf("mcrypt generic init failed")
	}

	bufferSize := len(blocks)
	if decrypt == true {
		for x := 0; x < bufferSize; x++ {
			C.mdecrypt_generic(td, unsafe.Pointer(&blocks[x]), C.int(1))
		}
	} else {
		for x := 0; x < bufferSize; x++ {
			C.mcrypt_generic(td, unsafe.Pointer(&blocks[x]), C.int(1))
		}
	}
	return nil
}

// CalculateCrc returns the Crc of a packet ready to be sent over the network,
// ignoring the Crc data part of it as it's done in the original nsca code
func (p *DataPacket) CalculateCrc(buffer []byte) uint32 {
	crcdPacket := make([]byte, len(buffer))
	copy(crcdPacket, buffer[0:4])
	copy(crcdPacket[8:], buffer[8:])
	return crc32.ChecksumIEEE(crcdPacket)
}
