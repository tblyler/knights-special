package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	// the max font size used for format-pretty spaces
	maxFontSize = 12

	// the base url for knights
	baseURL = "http://www.knightsrestaurants.com/specials/"
)

// regex to calculate format-pretty spaces per entry
var fontRegex = regexp.MustCompile("FONT-SIZE:([0-9]+)pt")

func main() {
	jsonify := false
	jsonPretty := false

	lunch := false
	dinner := false

	downtown := false
	annarbor := false
	jackson := false

	flag.BoolVar(&jsonify, "json", false, "output as JSON")
	flag.BoolVar(&jsonPretty, "jsonPretty", false, "output as pretty JSON")
	flag.BoolVar(&lunch, "lunch", false, "get the lunch special")
	flag.BoolVar(&dinner, "dinner", false, "get the dinner special")
	flag.BoolVar(&downtown, "downtown", false, "get the downtown special")
	flag.BoolVar(&annarbor, "annarbor", false, "get the Ann Arbor Dexter Rd special")
	flag.BoolVar(&jackson, "jackson", false, "get the Jackson special")
	flag.Parse()

	if jsonPretty {
		jsonify = true
	}

	if (lunch && dinner) || (!lunch && !dinner) {
		fmt.Fprintln(os.Stderr, "You must specify lunch or dinner, not multiples, nor neither")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if (downtown && (annarbor || jackson)) || (annarbor && (downtown || jackson)) || (jackson && (annarbor || downtown)) {
		fmt.Fprintln(os.Stderr, "You must specify only one location")
		flag.PrintDefaults()
		os.Exit(1)
	}

	url := baseURL
	if annarbor {
		if lunch {
			url += "lunch_kbr.html"
		} else if dinner {
			url += "dinner_kbr.html"
		} else {
			fmt.Fprintln(os.Stderr, "Unable to determine the meal to pull from for Ann Arbor Dexter Rd")
			flag.PrintDefaults()
			os.Exit(1)
		}
	} else if downtown {
		if lunch {
			url += "lunch_klm.html"
		} else if dinner {
			url += "dinner_klm.html"
		} else {
			fmt.Fprintln(os.Stderr, "Unable to determine the meal to pull from for downtown")
			flag.PrintDefaults()
			os.Exit(1)
		}
	} else if jackson {
		if lunch {
			url += "lunch_krj.html"
		} else if dinner {
			url += "dinner_krj.html"
		} else {
			fmt.Fprintln(os.Stderr, "Unable to determine the meal to pull from for downtown")
			flag.PrintDefaults()
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Unable to determine which location to pull from")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// run as a callback for graceful error handling
	err := func(url string, jsonify bool, jsonPretty bool) error {
		// pull the raw HTML
		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Got bad status on request: %d", resp.StatusCode)
		}

		// start the HTML tokenizer to parse the HTML pulled from the website
		tokenizer := html.NewTokenizer(resp.Body)
		if tokenizer.Err() != nil {
			return tokenizer.Err()
		}

		// this will store the output information
		output := []string{}

		// continue until /html
		for tokenizer.Token().Data != "html" {
			tokenType := tokenizer.Next()
			if tokenType == html.StartTagToken {
				token := tokenizer.Token()
				if token.Data == "td" {
					// keep track of the amount of spaces to prepend to the output string
					spaces := 0

					// go to the element that has text
					for inner := tokenizer.Next(); inner != html.TextToken; inner = tokenizer.Next() {
						if tokenizer.Err() != nil {
							return tokenizer.Err()
						}

						if !jsonify {
							// get the "font size" to determine the amount of spaces to use if this has a font attribute
							for _, attr := range tokenizer.Token().Attr {
								// use the predefined regex to check this attribute
								fontSizeMatch := fontRegex.FindStringSubmatch(attr.Val)

								// not a valid match, continue
								if fontSizeMatch == nil {
									continue
								}

								// trust the regex to not have a bad match for strcov.Atoi
								fontSize, _ := strconv.Atoi(fontSizeMatch[1])

								// assume that maxFontSize is correct
								spaces = maxFontSize - fontSize
								break
							}
						}
					}

					if !jsonify {
						// if this is the first entry, have no spaces regardless of the font value
						if len(output) == 0 {
							spaces = 0
						}
					}

					// prepend the amount of spaces to the output text and add it to the output slice
					output = append(output, strings.Repeat(" ", spaces)+strings.TrimSpace(string(tokenizer.Text())))
				}
			}
		}

		if jsonify {
			jsonOutput := ""
			if jsonPretty {
				jsonByte, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}

				jsonOutput = string(jsonByte)
			} else {
				jsonByte, err := json.Marshal(output)
				if err != nil {
					return err
				}

				jsonOutput = string(jsonByte)
			}

			fmt.Println(jsonOutput)
		} else {
			for _, line := range output {
				fmt.Println(line)
			}
		}

		return nil
	}(url, jsonify, jsonPretty)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
