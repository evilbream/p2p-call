package web

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"p2p-call/pkg/config"
	"p2p-call/pkg/connection"
	"p2p-call/pkg/system"
	"p2p-call/tmplt"
)

func StartWebInterface(wh *connection.WebsocketHandler) {
	port := system.GetWebPort()
	if port == "" {
		port = "8443" // temp default
	}
	// main page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("page").Parse(tmplt.HtmlPage))
		tmpl.Execute(w, nil)
	})

	// WebSocket to process audio and handler
	http.HandleFunc("/ws", wh.HandleWebsocketMessage)

	localIP := system.GetLocalIP()

	// generate self signed sertificate for HTTPS
	cert, err := config.GenerateSelfSignedCert()
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	server := &http.Server{
		Addr:      "0.0.0.0:" + port,
		TLSConfig: tlsConfig,
	}

	fmt.Printf("HTTPS Mobile access: https://%s:%s\n", localIP, port)
	fmt.Printf("HTTPS Local access: https://localhost:%s\n", port)

	log.Fatal(server.ListenAndServeTLS("", ""))

}
