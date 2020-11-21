package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/zplzpl/otpgateway/models"
)

const (
	providerID    = "solsms"
	channelName   = "SMS"
	addressName   = "Mobile number"
	maxAddresslen = 11
	maxOTPlen     = 6
	apiURL        = "https://api.kaleyra.io/v1/"
	statusOK      = "OK"
)

var reNum = regexp.MustCompile(`\+?([0-9]){8,15}`)

// sms is the default representation of the sms interface.
type sms struct {
	cfg *cfg
	h   *http.Client
}

type cfg struct {
	RootURL      string `json:"RootURL"`
	APIKey       string `json:"APIKey"`
	SID          string `json:"SID"`
	Sender       string `json:"Sender"`
	Timeout      int    `json:"Timeout"`
	MaxIdleConns int    `json:"MaxIdleConns"`
}

// solSMSAPIResp represents the response from solsms API.
type solSMSAPIResp struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// New returns an instance of the SMS package. cfg is configuration
// represented as a JSON string. Supported options are.
// {
// 	RootURL: "", // Optional root URL of the API,
// 	APIKey: "", // API Key,
// 	Sender: "", // Sender name
// 	Timeout: 5 // Optional HTTP timeout in seconds
// }
func New(jsonCfg []byte) (interface{}, error) {
	var c *cfg
	if err := json.Unmarshal(jsonCfg, &c); err != nil {
		return nil, err
	}
	if c.APIKey == "" || c.Sender == "" || c.SID == "" {
		return nil, errors.New("invalid APIKey or Sender or SID")
	}
	if c.RootURL == "" {
		c.RootURL = apiURL
	}

	c.RootURL = strings.TrimRight(c.RootURL, "/") + "/" + c.SID + "/messages"

	log.Println(c.RootURL)

	// Initialize the HTTP client.
	t := 5
	if c.Timeout != 0 {
		t = c.Timeout
	}
	h := &http.Client{
		Timeout: time.Duration(t) * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   1,
			ResponseHeaderTimeout: time.Second * time.Duration(t),
		},
	}

	return &sms{
		cfg: c,
		h:   h}, nil
}

// ID returns the Provider's ID.
func (s *sms) ID() string {
	return providerID
}

// ChannelName returns the Provider's name.
func (s *sms) ChannelName() string {
	return channelName
}

// AddressName returns the e-mail Provider's address name.
func (*sms) AddressName() string {
	return addressName
}

// ChannelDesc returns help text for the SMS verification Provider.
func (s *sms) ChannelDesc() string {
	return fmt.Sprintf(`
		We've sent a %d digit code in an SMS to your mobile.
		Enter it here to verify your mobile number.`, maxOTPlen)
}

// AddressDesc returns help text for the phone number.
func (s *sms) AddressDesc() string {
	return "Please enter your mobile number"
}

// ValidateAddress "validates" a phone number.
func (s *sms) ValidateAddress(to string) error {
	if !reNum.MatchString(to) {
		return errors.New("invalid mobile number")
	}
	return nil
}

// Push pushes out an SMS.
func (s *sms) Push(otp models.OTP, subject string, body []byte) error {

	var p = url.Values{}
	p.Set("sender", s.cfg.Sender)
	p.Set("to", otp.To)
	p.Set("body", string(body))

	// Make the request.
	req, err := http.NewRequest("POST", s.cfg.RootURL, strings.NewReader(p.Encode()))
	log.Println(p.Encode())
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("api-key", s.cfg.APIKey)
	log.Println(req)

	resp, err := s.h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response.
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// We now unmarshal the body.
	r := solSMSAPIResp{}
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	if r.Status != statusOK {
		return errors.New(r.Message)
	}
	return nil
}

// MaxAddressLen returns the maximum allowed length for the mobile number.
func (s *sms) MaxAddressLen() int {
	return maxAddresslen
}

// MaxOTPLen returns the maximum allowed length of the OTP value.
func (s *sms) MaxOTPLen() int {
	return maxOTPlen
}

// MaxBodyLen returns the max permitted body size.
func (s *sms) MaxBodyLen() int {
	return 140
}
