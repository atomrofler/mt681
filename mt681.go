package powermetermt681

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"

	"github.com/jacobsa/go-serial/serial"
)

var serialPortOptions serial.OpenOptions = serial.OpenOptions{
	PortName:              "/dev/ttyUSB0",
	BaudRate:              9600,
	DataBits:              8,
	StopBits:              1,
	ParityMode:            serial.PARITY_NONE,
	MinimumReadSize:       1,
	InterCharacterTimeout: 10,
}

// TODO
// # Stromverbrauch Total  77070100 010800	FF	650000018201621E52FF59  000000000517E31B        01
// # Zaehler Tarif 1???    77070100 010801	FF	0101621E52FF59          000000000517E31B        01
// # Zaehler Tarif 2???    77070100 010802	FF	0101621E52FF59          0000000000000000        01
// # Momentverbrauch Total 77070100 100700	FF	0101621B520055          0000021D                01
// # Stromverbrauch P1     77070100 240700	FF	0101621B520055          000000BC                01
// # Stromverbrauch P2     77070100 380700	FF	0101621B520055          000000A8                01
// # Stromverbrauch P3     77070100 4C0700	FF	0101621B520055          000000B9                01

type SmlDescription struct {
	id          string
	bezeichnung string
	laenge      int
}

type eHz struct {
	// Zeitstempel                time.Time
	WirkenergieGesamtBezug     int32
	WirkenergieTarif1Bezug     int32
	WirkenergieTarif2Bezug     int32
	WirkenergieGesamtLieferung int32
	WirkenergieTarif1Lieferung int32
	WirkenergieTarif2Lieferung int32
	WirkleistungTotal          int32
	WirkleistungPhase1         int32
	WirkleistungPhase2         int32
	WirkleistungPhase3         int32
}

// type SmartmeterConfig struct {
// 	Device   string `json:device`
// 	Loglevel string `json:loglevel`
// }

var sml_messages = []SmlDescription{
	{
		id:          "0100010800",
		bezeichnung: "WirkenergieGesamtBezug",
		laenge:      16,
	}, {
		id:          "0100010801",
		bezeichnung: "WirkenergieTarif1Bezug",
		laenge:      16,
	}, {
		id:          "0100010802",
		bezeichnung: "WirkenergieTarif2Bezug",
		laenge:      16,
	}, {
		id:          "0100020800",
		bezeichnung: "WirkenergieGesamtLieferung",
		laenge:      16,
	}, {
		id:          "0100020801",
		bezeichnung: "WirkenergieTarif1Lieferung",
		laenge:      16,
	}, {
		id:          "0100020802",
		bezeichnung: "WirkenergieTarif2Lieferung",
		laenge:      16,
	}, {
		id:          "0100100700",
		bezeichnung: "WirkleistungTotal",
		laenge:      8,
	}, {
		id:          "0100240700",
		bezeichnung: "WirkleistungPhase1",
		laenge:      8,
	}, {
		id:          "0100380700",
		bezeichnung: "WirkleistungPhase2",
		laenge:      8,
	}, {
		id:          "01004C0700",
		bezeichnung: "WirkleistungPhase3",
		laenge:      8,
	},
}

func New(device string, loglevel string) error {
	// TODO klog
	// https://pkg.go.dev/k8s.io/klog/v2
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING") // [WARNING|ERROR|INFO|FATAL]
	flag.Parse()

	// config, err := getConfig()

	fmt.Println("configure device " + device)
	setupDevice(device)
	return nil
}

// func getConfig() (SmartmeterConfig, error) {
// 	config := SmartmeterConfig{}

// 	path, err := os.Getwd()
// 	if err != nil {
// 		log.Println(err)
// 	}

// 	fmt.Println("Open config smartmeter.json in " + path) // for example /home/user
// 	content_config, err := ioutil.ReadFile("smartmeter.json")
// 	if err != nil {
// 		fmt.Println("error open configfile")
// 		fmt.Println(err)
// 		return config, errors.New("error reading smartmeter.json")
// 	}
// 	fmt.Println(string(content_config[:]))

// 	err = json.Unmarshal([]byte(content_config), &config)
// 	if err != nil {
// 		return config, errors.New("cant unmarshal smartmeter.json")
// 	}

// 	if config.Device == "" {
// 		return config, errors.New("No device in smartmeter.json ")
// 	}

// 	return config, nil
// }

func setupDevice(device string) {
	glog.Info("Setup Device: " + device)
	serialPortOptions.PortName = device
}

func GetSerialOptions() {
	log.Println(serialPortOptions)
}

func openPort() (io.ReadWriteCloser, error) {
	port, err := serial.Open(serialPortOptions)
	// fmt.Printf("%+v", serialPortOptions)
	if err != nil {
		log.Println("Error open Port")
		// message := "testmessage"
		// glog.Error(message)
		return nil, errors.New("Error on serial.open")
	}
	// glog.Info("test")
	// glog.Warningln("WARnING!")
	return port, nil
}

func GeteHzData() eHz {
	data, _ := readMessages()
	return data
}

func readMessages() (eHz, error) {
	port, err := openPort()
	if err != nil {
		return eHz{}, err
	}

	var error error

	buffer := make([]byte, 950)
	_, err = io.ReadAtLeast(port, buffer, 950)

	if err != nil {
		log.Println("errr")
	}

	// buf = buf[:data]
	messages_raw := strings.ToUpper(hex.EncodeToString(buffer))

	// log.Println("RX: ", messages_raw)

	if isMessageComplete(messages_raw) != nil {
		log.Println(err)
		error = err
		return eHz{}, error
	}

	message_raw := getOneMessageFromRawData(messages_raw)
	// https://www.makerconnect.de/index.php?threads/smart-meter-auslesen.3072/

	filtered_messages := getFilteredMessages(message_raw)

	var ehz eHz
	for i := 0; i < len(filtered_messages); i++ {
		// log.Println("parse msg: ", filtered_messages[i][0])
		sml_id, sml_value := parseMessage(filtered_messages[i][0])

		var msgdesc SmlDescription = getMessageDescriptionById(sml_id)

		if msgdesc.id != "" {
			// log.Println("Desc:        ", msgdesc.bezeichnung)
			// log.Println("Message ID:  ", sml_id)
			// log.Println("Message VAL: ", sml_value)

			calcvalue := calcPower(sml_value, msgdesc.laenge)
			// log.Println("CALC VALUE: ", calcvalue)
			// ehz = append(ehz, eHz{
			// 	name: msgdesc.bezeichnung,
			// 	wert: calcvalue,
			// })

			if msgdesc.bezeichnung == "WirkenergieGesamtBezug" {
				ehz.WirkenergieGesamtBezug = calcvalue
			}
			if msgdesc.bezeichnung == "WirkenergieTarif1Bezug" {
				ehz.WirkenergieTarif1Bezug = calcvalue
			}
			if msgdesc.bezeichnung == "WirkenergieTarif2Bezug" {
				ehz.WirkenergieTarif2Bezug = calcvalue
			}
			if msgdesc.bezeichnung == "WirkenergieGesamtLieferung" {
				ehz.WirkenergieGesamtLieferung = calcvalue
			}
			if msgdesc.bezeichnung == "WirkenergieTarif1Lieferung" {
				ehz.WirkenergieTarif1Lieferung = calcvalue
			}
			if msgdesc.bezeichnung == "WirkenergieTarif2Lieferung" {
				ehz.WirkenergieTarif2Lieferung = calcvalue
			}
			if msgdesc.bezeichnung == "WirkleistungTotal" {
				ehz.WirkleistungTotal = calcvalue
			}
			if msgdesc.bezeichnung == "WirkleistungPhase1" {
				ehz.WirkleistungPhase1 = calcvalue
			}
			if msgdesc.bezeichnung == "WirkleistungPhase2" {
				ehz.WirkleistungPhase2 = calcvalue
			}
			if msgdesc.bezeichnung == "WirkleistungPhase3" {
				ehz.WirkleistungPhase3 = calcvalue
			}

		} else {
			log.Println("unbekannte id")
		}

	}
	defer port.Close()
	return ehz, errors.New("err")
}

func parseMessage(message string) (string, string) {
	regx := regexp.MustCompile("^7707(.{10})FF(:?65000101.201621E52FF59|0101621E52FF59|0101621B520055)(.{8,16}?)01$")
	parsed_message := regx.FindStringSubmatch(message)
	sml_id := ""
	sml_value := ""
	if len(parsed_message) > 0 {
		sml_id = parsed_message[1]
		// sml_unknown := parsed_message[2]
		sml_value = parsed_message[3]
	}
	return sml_id, sml_value
}

func calcPower(hexvalue string, length int) int32 {
	value_uint, _ := strconv.ParseUint(hexvalue, 16, 32)
	int32value := int32(value_uint)

	// bei laengen von 16 rechnen wir durch 10000, um kwh zu bekommen
	if length == 16 {
		int32value /= int32(10000)
	}
	return int32value
}

func getMessageDescriptionById(id string) SmlDescription {
	var smlDescription SmlDescription
	for i := range sml_messages {
		if sml_messages[i].id == id {
			smlDescription = sml_messages[i]
			break
		} else {
			smlDescription.id = ""
		}
	}
	return smlDescription
}

// 1B1B1B1B - Start Escape Zeichenfolge
// 01010101 - Start Übertragung Version 1

// 	1B1B1B1B - Ende Escape Zeichenfolge
// 1A - Ende der Nachricht
// 00 - Anzahl Erweiterungsbyte
// 8F28 - Checksumme gesamte Nachricht (CCITT-CRC16)
func isMessageComplete(message string) error {
	match, _ := regexp.MatchString("1B1B1B1B01010101.{886}001B1B1B1B1A00", message)
	if match {
		return nil
	} else {
		errmsg := ""
		if len(message) > 0 {
			len := len(message)
			errmsg = "Gesamt Message nicht komplett. Laenge " + strconv.Itoa(len) + ". Abbruch\nMessage:\n" + message
		}
		return errors.New(errmsg)
	}
}

func getFilteredMessages(allmessages string) [][]string {
	regx := regexp.MustCompile("(7707.{10}FF(?:65000101.201621E52FF59|0101621E52FF59|0101621B520055).{8,16}?01)")
	return regx.FindAllStringSubmatch(allmessages, -1)
}

func getOneMessageFromRawData(message string) string {
	regx := regexp.MustCompile("(1B1B1B1B01010101.{886}001B1B1B1B1A00)")
	matches := regx.FindStringSubmatch(message)
	return matches[0]
}
