// Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
//  Professor: Fernando Dotti  (https://fldotti.github.io/)
// Uso p exemplo:
//   go run usaDIMEX.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
//   go run usaDIMEX.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
//   go run usaDIMEX.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
// ----------
// LANCAR N PROCESSOS EM SHELL's DIFERENTES, UMA PARA CADA PROCESSO.
// para cada processo fornecer: seu id único (0, 1, 2 ...) e a mesma lista de processos.
// o endereco de cada processo é o dado na lista, na posicao do seu id.
// no exemplo acima o processo com id=1  usa a porta 6001 para receber e as portas
// 5000 e 7002 para mandar mensagens respectivamente para processos com id=0 e 2
// -----------
// Esta versão supõe que todos processos tem acesso a um mesmo arquivo chamado "mxOUT.txt"
// Todos processos escrevem neste arquivo, usando o protocolo dimex para exclusao mutua.
// Os processos escrevem "|." cada vez que acessam o arquivo.   Assim, o arquivo com conteúdo
// correto deverá ser uma sequencia de
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// etc etc ...     ....  até o usuário interromper os processos (ctl c).
// Qualquer padrao diferente disso, revela um erro.
//      |.|.|.|.|.||..|.|.|.  etc etc  por exemplo.
// Se voce retirar o protocolo dimex vai ver que o arquivo poderá entrelacar
// "|."  dos processos de diversas diferentes formas.
// Ou seja, o padrão correto acima é garantido pelo dimex.
// Ainda assim, isto é apenas um teste.  E testes são frágeis em sistemas distribuídos.

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
	dmx := DIMEX.NewDIMEX(addresses, id, true) // Inicializa o módulo DIMEX	
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

		snapshotID := 1
		go func() {
			for {
				time.Sleep(1 * time.Second) // Adjust the interval as needed
				fmt.Println("[ APP id: ", id, " INITIATING SNAPSHOT ", snapshotID, " ]")
				dmx.InitiateSnapshot(snapshotID)
				snapshotID++
			}
		}()
	
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