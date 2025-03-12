package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	"gitlab.com/iglou.eu/goulc/http/client"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

const (
	Username       = "doom"
	Password       = "slayer"
	BasicAuthValue = "Basic ZG9vbTpzbGF5ZXI="
)

func main() {
	// Create test server
	ts := Test_DoomServer()
	defer ts.Close()

	// You can use the client.OptDefault, but it provide a "safe" configuration,
	// So we can't use it in this non-secure environment.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opt := client.Options{
		OnlyHTTPS: false,
		Timeout:   time.Second * 5,
	}

	// Create an HTTP client without authentication
	httpClient, err := client.New(ctx, ts.URL, nil, &opt, slog.Default())
	if err != nil {
		panic(err)
	}

	// #01 Get request to / without Unmarshaler
	res, err := httpClient.Do(http.MethodGet, nil, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nTest #01: That work like a hell\nStatus: %v; Body: %s\n", res.Status, res.Body)

	// #02 Get request to / with Unmarshaler
	// view DoomResponse struct that implements client.Unmarshaler
	res, err = httpClient.Do(http.MethodGet, nil, &DoomResponse{})
	if err != nil {
		panic(err)
	}

	doomRes := res.BodyUml.(*DoomResponse)
	fmt.Printf("\nTest #02: Like you can see, my body was unmarshaled\nStatus: %v; Body: %#v\n", doomRes.Status, doomRes)

	// #03 Get request to /demons
	// We can update the uri path or create a new client with the path extension
	// For this example, we will create a new "child client"
	httpClientDeamons := httpClient.NewChild("/demons")
	res, err = httpClientDeamons.Do(http.MethodGet, nil, &DoomResponse{})
	if err != nil {
		panic(err)
	}

	doomRes = res.BodyUml.(*DoomResponse)
	fmt.Printf("\nTest #03: Ho nooo, you are not authenticated, the door stay closed\nStatus: %v; Body: %#v\n", doomRes.Status, doomRes)

	// #04 Get request to /demons with authentication
	// We need to create an authentification that support auth.Authenticator interface
	auth, err := auth.NewBasic(Username, Password)
	if err != nil {
		panic(err)
	}

	// We can set the authentication on the client
	httpClientDeamons.Auth = &auth

	// And now, we can make the request with authentication
	res, err = httpClientDeamons.Do(http.MethodGet, nil, &DoomResponse{})
	if err != nil {
		panic(err)
	}

	doomRes = res.BodyUml.(*DoomResponse)
	fmt.Printf("\nTest #04: And we are authenticated, the door open\nStatus: %v; Body: %#v\n", doomRes.Status, doomRes)

	// #05 Post request to /weapons
	// We want to take a powerful weapon, to do this, we need to send JSON body
	// But, who want to write and manage mashalization for every request ?!
	// Not me, so we will use the client.Marshaler interface with DoWithMarshal

	// First, create a new client with the path extension
	httpClientWeapons := httpClient.NewChild("/weapons")

	// Create a new DoomWeapon
	weapon := DoomWeapon{
		Weapon: "BFG 9000",
		Power:  9000,
	}

	// Create a new DoomResponse
	doomResp := new(DoomResponse)

	// And now, we can make the request with marshalling. Unlike previous
	// examples, we use an existing DoomResponse variable. This is more
	// convenient and avoid type-assertion stuff
	res, err = httpClientWeapons.DoWithMarshal(
		http.MethodPost, &weapon, doomResp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nTest #05: And we have now a BFG, let's ROCK !\nStatus: %v; Body: %#v\n", doomRes.Status, doomRes)

	// #06 into the response, you can see some metrics
	// Let's try a bad endpoint after a redirect
	httpClientRedirect := httpClient.NewChild("/redirect")
	httpClientRedirect.Options.Follow = true
	httpClientRedirect.Options.MaxRedirect = 2
	res, err = httpClientRedirect.Do(http.MethodGet, nil, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nTest #06: Look there metrics in your face.\nStatus: %v\nResponseTime: %v\nTrace: %#v\nErrorRate: %v%%\n", res.Status, res.ResponseTime, res.Trace, res.ErrorRate)

	// #07 Close the client
	// To release clients resources, you need to call Close() method, that will
	// release all resources used by the client and its children in cascade
	//
	// The main ctx close, are used to cancel request with the http.
	// NewRequestWithContext At the next Do/DoWithMarshal, the client will be
	// closed automatically if not closed manually
	err = httpClient.Close()
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nTest #07: Client closed successfully, and child are closed too in cascade.\n> Main client: %#v\n> Child client: %#v\n", httpClient, httpClientWeapons)

}

type DoomResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Info    string `json:"info"`
}

// To check that the struct implements the client.Unmarshaler interface
// Like you can see, this is the pointer to the struct which implements the interface
// Because, Unmarshal need to populates the receiver
var _ client.Unmarshaler = (*DoomResponse)(nil)

func (d *DoomResponse) Name() string {
	return "test.Doom"
}

// To keep the demo simple, we only unmarshal json, but you can unmarshal
// anything you want with any required processing
func (d *DoomResponse) Unmarshal(_ int, _ http.Header, body []byte) error {
	return json.Unmarshal(body, d)
}

type DoomWeapon struct {
	Weapon string `json:"weapon"`
	Power  int    `json:"power"`
}

// Like the previous example, we will implement the client.Marshaler interface
// That can implemented on the pointer or the struct, feel free to choose.
var _ client.Marshaler = DoomWeapon{}

func (d DoomWeapon) Name() string {
	return "test.DoomWeapon"
}

// Required to inform the client of the content type\
func (d DoomWeapon) ContentType() string {
	return "application/json"
}

func (d DoomWeapon) Marshal() ([]byte, error) {
	return json.Marshal(d)
}

// Just a helper function to create a test server
func Test_DoomServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var response DoomResponse

		switch r.URL.Path {
		case "", "/":
			response = DoomResponse{
				Status:  200,
				Message: "Success",
				Info:    "Welcome to Hell !",
			}
		case "/demons":
			if r.Header.Get("Authorization") != BasicAuthValue {
				response = DoomResponse{
					Status:  401,
					Message: "Unauthorized",
				}
				w.WriteHeader(http.StatusUnauthorized)
				break
			}
			response = DoomResponse{
				Status:  200,
				Message: "Success",
				Info:    "The doom gate are open, and some cacodemons fly around you.",
			}
		case "/weapons":
			if r.Method != http.MethodPost {
				response = DoomResponse{
					Status:  405,
					Message: "Method not allowed",
				}
				w.WriteHeader(http.StatusMethodNotAllowed)
				break
			}
			response = DoomResponse{
				Status:  200,
				Message: "Success",
				Info:    "Flexing your muscles, you grab a BFG, and you are ready to kill some demons.",
			}
		case "/redirect":
			w.Header().Set("Location", r.URL.String()+"/out")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		default:
			response = DoomResponse{
				Status:  404,
				Message: "Endpoint not found",
			}
			w.WriteHeader(http.StatusNotFound)
		}

		json.NewEncoder(w).Encode(response)
	}))
}
