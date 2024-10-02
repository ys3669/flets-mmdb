package main

import (
	"encoding/csv"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/joho/godotenv"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType: "flets-NGN-DB",
			RecordSize:   24,
		},
	)

	TOKEN := os.Getenv("ipinfo_token")

	client := ipinfo.NewClient(nil, nil, TOKEN)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range []string{"flets-routes.csv"} {
		fh, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}

		r := csv.NewReader(fh)
		r.Comment = '#'

		// first line
		r.Read()

		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			if len(row) != 2 {
				log.Fatalf("unexpected CSV rows: %v", row)
			}

			type_number, err := strconv.Atoi(row[0])
			if err != nil {
				log.Fatal(err)
			}

			ip, network, err := net.ParseCIDR(row[1])
			if err != nil {
				log.Fatal(err)
			}

			ipinfo, ipinfo_err := client.GetIPInfo(ip)
			if ipinfo_err != nil {
				log.Fatal(err)
			}

			record := mmdbtype.Map{}

			if type_number != 0 {
				record["type_number"] = mmdbtype.Uint32(type_number)

				firstDigit := strconv.Itoa(type_number)[0]

				switch firstDigit {
				case '1':
					record["zone"] = mmdbtype.String("east")
				case '2':
					record["zone"] = mmdbtype.String("west")
				default:
					record["zone"] = mmdbtype.String("unknown")
				}

				secondDigit := strconv.Itoa(type_number)[1] //アドレス帯の情報
				thirdDigit := strconv.Itoa(type_number)[2]  //利用用途

				switch secondDigit {
				case '1':
					record["address_range"] = mmdbtype.String("IPNetwork") //IP通信網
					switch thirdDigit {
					case '1':
						record["usecase"] = mmdbtype.String("PPPoE infrastructure") //*11* PPPoE接続基盤
					default:
						record["usecase"] = mmdbtype.String("unknown")
					}
				case '2':
					record["address_range"] = mmdbtype.String("IPNetwork") //IP通信網
					switch thirdDigit {
					case '1':
						record["usecase"] = mmdbtype.String("IPoE infrastructure") //*12* IPoE基盤
					default:
						record["usecase"] = mmdbtype.String("unknown")
					}
				case '3':
					record["address_range"] = mmdbtype.String("IPNetwork")
					switch thirdDigit {
					case '1':
						record["usecase"] = mmdbtype.String("In-net turnaround infrastructure") //*13* 網内折り返し基盤
					default:
						record["usecase"] = mmdbtype.String("unknown")
					}
				case '4':
					record["address_range"] = mmdbtype.String("Service providers") //接続事業者
					switch thirdDigit {
					case '1':
					case '2':
						record["usecase"] = mmdbtype.String("IPoE") //*14* IPv6インターネット接続(IPoE)
					default:
						record["usecase"] = mmdbtype.String("unknown")
					}
				default:
					record["address_range"] = mmdbtype.String("unknown")
					record["usecase"] = mmdbtype.String("unknown")
				}
			}

			record["autonomous_system_organization"] = mmdbtype.String(ipinfo.Org)

			err = writer.Insert(network, record)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	fh, err := os.Create("flets.mmdb")
	if err != nil {
		log.Fatal(err)
	}

	_, err = writer.WriteTo(fh)
	if err != nil {
		log.Fatal(err)
	}
}
