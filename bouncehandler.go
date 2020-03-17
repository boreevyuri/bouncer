package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"gopkg.in/erizocosmico/go-bouncespy.v1"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"strings"
)

var (
//EmailRegexp = regexp.MustCompile(`^[\w\d\.\_\%\+\-]+@([\w\d\.\-]+\.\w{2,16})$`)
//yahooRegexp = regexp.MustCompile(`^To\:\ [\w\d\.\_\%\+\-]+@([\w\d\.\-]+\.\w{2,16})$`)
)

func main() {
	dir := os.Args[1]
	files, err := ioutil.ReadDir(dir)

	if err == nil {

		for _, file := range files {
			data, err := ioutil.ReadFile(dir + "/" + file.Name())
			if err != nil {
				fmt.Println("File reading error", err)
				return
			}

			var realRcptAddr = ""
			var reason = ""

			r := strings.NewReader(string(data))
			m, err := mail.ReadMessage(r)
			if err != nil {
				log.Fatal(err)
			}

			header := m.Header

			// для начала ищем заголовок X-Failed-Recipients. Если его нет - продолжаем обработку
			failedRcptAddr := header.Get("X-Failed-Recipients")
			if len(failedRcptAddr) > 0 {
				realRcptAddr = failedRcptAddr
			} else {
				switch hFrom := header.Get("from"); hFrom {
				case "MAILER-DAEMON@yahoo.com":
					realRcptAddr, reason = getYahooData(m)

				}
			}

			body, err := ioutil.ReadAll(m.Body)
			if err != nil {
				log.Fatal(err)
			}

			result := bouncespy.Analyze(header, body)
			if len(reason) == 0 {
				reason = string(result.Reason)
			}
			if len(realRcptAddr) == 0 {
				realRcptAddr = findOriginalRecipient(body)
			}

			//if len(realRcptAddr) > 0 {
			fmt.Printf("%s | %s | Type: %d| file: %s\n", realRcptAddr, reason, result.Type, file.Name())
			//}
			//if result.Type > 0 {
			//fmt.Printf("%s | %s | Type: %d| file: %s\n", realRcptAddr, result.Reason, result.Type, file.Name())
			//}
			//fmt.Printf("%s | %s | Type: %d| file: %s\n", realRcptAddr, result.Reason, result.Type, file.Name())
		}
	}
}

func findOriginalRecipient(body []byte) string {
	rcptAddr := ""
	lns := strings.Split(strings.ToLower(string(body)), "\n")
	numLines := len(lns)
	var lines = make([]string, numLines)
	for i, ln := range lns {
		lines[numLines-i-1] = ln
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "original-recipient:") {
			parts := strings.Split(line, ";")
			if len(parts) > 1 {
				rcptAddr = strings.Trim(parts[1], "<>")
			}
		}
	}
	return rcptAddr
}

//func getDomain(h string) (domain string, err error) {
//	parser := new(mail.AddressParser)
//	fromAddr, _ := parser.Parse(h)
//	domain, err = getHostnameFromEmail(fromAddr.Address)
//	return domain, err
//}

//func getHostnameFromEmail(email string) (string, error) {
//	matches := EmailRegexp.FindAllStringSubmatch(email, -1)
//	if len(matches) == 1 && len(matches[0]) == 2 {
//		return matches[0][1], nil
//	}
//	return "", errors.New("invalid email address")
//}

func getYahooData(m *mail.Message) (string, string) {

	switch bodyEnc := m.Header.Get("Content-Transfer-Encoding"); bodyEnc {
	case "base64":
		return parseYahooBase64(m.Body)
	case "quoted-printable":
		return parseYahooQuoted(m.Body)
	default:
		return parseYahooNotice(m)
	}

}

//func parseYahooBounce(m *mail.Message) (string, string, err error) {
//	rcptAddr := ""
//	reason := "delivery failed"
//	// проверяем заголовок Subject. Если есть - просто достаем To:
//	if m.Header.Get("subject") == "failure notice" {
//
//	}
//}

func parseYahooBase64(b io.Reader) (string, string) {
	rcptAddr := ""
	reason := "delivery failed"
	r := bufio.NewReader(b)
	dec := base64.NewDecoder(base64.StdEncoding, r)
	scanner := bufio.NewScanner(dec)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "To: ") {
			rcptAddr = strings.TrimPrefix(scanner.Text(), "To: ")
			break
		}
	}
	return rcptAddr, reason
}

func parseYahooNotice(m *mail.Message) (string, string) {
	if m.Header.Get("subject") != "failure notice" {
		return "unable to parse mail from Yahoo", "unknown reason"
	}
	rcptAddr := ""
	reason := ""
	found := false
	scanner := bufio.NewScanner(m.Body)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if !found {
			if strings.Contains(scanner.Text(), "@") {
				rcptAddr = strings.Trim(scanner.Text(), "<>:")
				found = true
			}
		} else {
			reason = scanner.Text()
			break
		}
	}

	return rcptAddr, reason
}

func parseYahooQuoted(b io.Reader) (string, string) {
	rcptAddr := ""
	reason := "delivery failed"
	scanner := bufio.NewScanner(b)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "To: ") {
			rcptAddr = strings.TrimPrefix(scanner.Text(), "To: ")
			break
		}
	}
	return rcptAddr, reason
}
