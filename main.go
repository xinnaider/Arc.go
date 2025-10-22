package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func workerLoop(workerID int, baseURL string, queue string, client *http.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		// pega o próximo ID em /jobs/
		resp, err := client.Get(fmt.Sprintf("%s/jobs/%s/ack", strings.TrimRight(baseURL, "/"), queue))
		
		if err != nil {
			log.Printf("[worker %d] erro ao buscar /jobs/: %v", workerID, err)
			time.Sleep(2 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNoContent {
			log.Printf("[worker %d] sem jobs (204). aguardando 10s…", workerID)
			time.Sleep(10 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			jobID := strings.TrimSpace(string(body))
			if jobID == "" {
				continue
			}

			runURL := fmt.Sprintf("%s/jobs/%s/run", strings.TrimRight(baseURL, "/"), jobID)
			runResp, err := client.Get(runURL)
			if err != nil {
				log.Printf("[worker %d] erro ao chamar %s: %v", workerID, runURL, err)
				time.Sleep(1 * time.Second)
				continue
			}
			io.Copy(io.Discard, runResp.Body)
			runResp.Body.Close()

			log.Printf("[worker %d] job %s disparado (status %d)", workerID, jobID, runResp.StatusCode)
			continue
		}

		time.Sleep(2 * time.Second)
	}
}

func main() {
	workers := flag.Int("workers", 3, "Número de workers concorrentes")
	baseURL := flag.String("url", "https://localhost", "Base URL da API (ex.: https://localhost)")
	queue := flag.String("queue", "default", "Nome da fila")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout HTTP")
	insecure := flag.Bool("insecure", true, "Aceitar TLS inseguro")

	flag.Parse()
	log.Printf("Iniciando com %d worker(s) na fila '%s' para %s (timeout %s, insecure=%v)", *workers, *queue, *baseURL, timeout.String(), *insecure)

	transport := &http.Transport{}
	if *insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} 
	}

	client := &http.Client{
		Timeout:   *timeout,
		Transport: transport,
	}

	var wg sync.WaitGroup
	for i := 1; i <= *workers; i++ {
		wg.Add(1)
		go workerLoop(i, *baseURL, *queue, client, &wg)
	}
	wg.Wait()
}