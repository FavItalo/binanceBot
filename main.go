package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var assets = []string{"btcusdt", "ethusdt", "bnbusdt", "xrpusdt", "solusdt", "adausdt"}

// Estrutura para a resposta do ticker
type TickerMessage struct {
	EventType        string      `json:"e"`
	EventTime        json.Number `json:"E"`
	Symbol           string      `json:"s"`
	PriceChange      json.Number `json:"p"`
	PriceChangePct   json.Number `json:"P"`
	WeightedAvgPrice json.Number `json:"w"`
	PrevClosePrice   json.Number `json:"x"`
	LastPrice        json.Number `json:"c"`
	LastQty          json.Number `json:"Q"`
	BidPrice         json.Number `json:"b"`
	BidQuantity      json.Number `json:"B"`
	AskPrice         json.Number `json:"a"`
	AskQuantity      json.Number `json:"A"`
	LastTime         json.Number `json:"C"`
}

// Estruturas para armazenar os dados e o histórico de preços
var (
	tickerData     = make(map[string]TickerMessage) // Dados atuais
	prevPrices     = make(map[string]float64)       // Preços anteriores para comparação
	tickerUpdateCh = make(chan TickerMessage)       // Canal para atualizações de dados
	mutex          = sync.Mutex{}
)

// Função para monitorar cada ativo em um WebSocket separado
func monitorAsset(symbol string) {
	url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@ticker", symbol)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("Erro ao conectar ao WebSocket para %s: %v", symbol, err)
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Erro ao ler mensagem do %s: %v", symbol, err)
			return
		}

		var ticker TickerMessage
		err = json.Unmarshal(message, &ticker)
		if err != nil {
			log.Printf("Erro ao desserializar JSON para %s: %v", symbol, err)
			continue
		}

		// Normaliza o símbolo para letras minúsculas antes de enviar para o canal
		ticker.Symbol = strings.ToLower(ticker.Symbol)
		tickerUpdateCh <- ticker
	}
}

// Função para limpar o terminal
func clearTerminal() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// Função para exibir os dados dos ativos em uma tabela
func printTable() {
	clearTerminal()

	// Cabeçalho da tabela
	fmt.Printf("\n%-10s | %-12s | %-10s | %-10s | %-10s | %-10s\n", "Pair", "Último Preço", "Bid", "Ask", "Mudança", "Mudança (%)")
	fmt.Println("------------------------------------------------------------------------------------------")

	// Usa o mutex para proteger o acesso ao mapa durante a leitura
	mutex.Lock()
	defer mutex.Unlock()

	for _, symbol := range assets {
		// Normaliza o símbolo para letras minúsculas ao acessar o mapa
		symbolLower := strings.ToLower(symbol)
		data, exists := tickerData[symbolLower]
		if exists {
			// Converte os valores para float para manipulação numérica
			lastPrice, err1 := data.LastPrice.Float64()
			bidPrice, err2 := data.BidPrice.Float64()
			askPrice, err3 := data.AskPrice.Float64()
			priceChange, err4 := data.PriceChange.Float64()
			priceChangePct, err5 := data.PriceChangePct.Float64()

			// Comparação do último preço com o preço anterior
			priceColor := "\033[0m" // Cor padrão
			if err1 == nil {
				prevPrice, hasPrev := prevPrices[symbolLower]
				if hasPrev {
					if lastPrice > prevPrice {
						priceColor = "\033[32m" // Verde para alta
					} else if lastPrice < prevPrice {
						priceColor = "\033[31m" // Vermelho para baixa
					}
				}
				// Atualiza o preço anterior
				prevPrices[symbolLower] = lastPrice
			}

			// Exibe a linha com os dados de cada ativo, ou mensagens de erro em caso de falha na conversão
			if err1 == nil && err2 == nil && err3 == nil && err4 == nil && err5 == nil {
				fmt.Printf("%-10s | %s%-12.2f\033[0m | %-10.2f | %-10.2f | %-10.2f | %-10.2f%%\n",
					data.Symbol, priceColor, lastPrice, bidPrice, askPrice, priceChange, priceChangePct)
			} else {
				fmt.Printf("%-10s | Erro de Conversão\n", symbol)
			}
		} else {
			fmt.Printf("%-10s | %-12s | %-10s | %-10s | %-10s | %-10s\n", symbol, "-", "-", "-", "-", "-")
		}
	}
}

// Função principal que gerencia o canal de atualizações e impressão
func main() {
	// Inicia a monitoria de cada ativo
	for _, asset := range assets {
		go monitorAsset(asset)
	}

	// Goroutine para processar atualizações do canal e atualizar o tickerData
	go func() {
		for update := range tickerUpdateCh {
			// Trava o acesso ao mapa para atualizar tickerData
			mutex.Lock()
			tickerData[update.Symbol] = update
			mutex.Unlock()

			// Log para confirmar a atualização do mapa
			log.Printf("Atualização para %s armazenada em tickerData: %+v", update.Symbol, update)

			printTable() // Imprime a tabela ao receber uma nova atualização
		}
	}()

	// Mantém o programa em execução
	select {}
}
