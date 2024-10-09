//   go run useDIMEX-f.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002
//   go run useDIMEX-f.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002
//   go run useDIMEX-f.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002

package main

import (
	"SD/DIMEX"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Please specify at least one address:port!")
		return
	}

	id, _ := strconv.Atoi(os.Args[1])   // Identificador do processo (0, 1, 2, ...)
	addresses := os.Args[2:]            // Endereços dos processos
	violateSC := false   // Ativar falha de violação da SC
    blockRespOK := false // Ativar falha de bloqueio de resposta

    dmx := DIMEX.NewDIMEX(addresses, id, true, violateSC, blockRespOK)  // Inicializa o módulo DIMEX	
		// Ensure the snapshots directory exists
		if _, err := os.Stat("./snapshots"); os.IsNotExist(err) {
			err := os.Mkdir("./snapshots", 0755)
			if err != nil {
				fmt.Println("Error creating snapshots directory:", err)
				return
			}
		}

		// Abre o arquivo compartilhado "mxOUT.txt"
		file, err := os.OpenFile("./mxOUT.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()
	
		// Aguarda para sincronizar com os outros processos
		time.Sleep(3 * time.Second)

		// if id == 0 {
			snapshotID := 1
			go func() {
				for {
					time.Sleep(200 * time.Millisecond) // Adjust the interval as needed
					fmt.Println("[ APP id: ", id, " INITIATING SNAPSHOT ", snapshotID, " ]")
					dmx.InitiateSnapshot(snapshotID)
					snapshotID++
				}
			}()
		// }
	
		for {
			// 1. Solicita entrada na Seção Crítica (SC)
			fmt.Println("[ APP id: ", id, " PEDE ACESSO À SC ]")
			dmx.Req <- DIMEX.ENTER // Envia solicitação de entrada
	
			// 2. Espera até obter acesso à SC
			<-dmx.Ind // Espera o módulo DIMEX dar permissão para entrar na SC
			fmt.Println("[ APP id: ", id, " ENTROU NA SC ]")
	
			// 3. Escreve no arquivo "mxOUT.txt" o padrão |.
			_, err = file.WriteString("|")
			if err != nil {
				fmt.Println("Error writing to file:", err)
				return
			}
			time.Sleep(500 * time.Millisecond) // Simula algum processamento dentro da SC
	
			_, err = file.WriteString(".")
			if err != nil {
				fmt.Println("Error writing to file:", err)
				return
			}
			fmt.Println("[ APP id: ", id, " SAIU DA SC ]")
	
			// 4. Libera a Seção Crítica (SC)
			dmx.Req <- DIMEX.EXIT // Informa ao módulo DIMEX que o processo saiu da SC
	
			// Tempo de espera antes de tentar novamente
			time.Sleep(2 * time.Second) // Pequeno atraso para evitar "busy waiting"
		}
	}