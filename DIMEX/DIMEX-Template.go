/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
    Professor: Fernando Dotti  (https://fldotti.github.io/)
    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
    Semestre 2023/1
	Aspectos a observar:
	   mapeamento de módulo para estrutura
	   inicializacao
	   semantica de concorrência: cada evento é atômico
	   							  módulo trata 1 por vez
	Q U E S T A O
	   Além de obviamente entender a estrutura ...
	   Implementar o núcleo do algoritmo ja descrito, ou seja, o corpo das
	   funcoes reativas a cada entrada possível:
	   			handleUponReqEntry()  // recebe do nivel de cima (app)
				handleUponReqExit()   // recebe do nivel de cima (app)
				handleUponDeliverRespOk(msgOutro)   // recebe do nivel de baixo
				handleUponDeliverReqEntry(msgOutro) // recebe do nivel de baixo
*/
package DIMEX

import (
	PP2PLink "SD/PP2PLink"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type State int // enumeracao dos estados possiveis de um processo
const (
	noMX State = iota
	wantMX
	inMX
)

type dmxReq int // enumeracao dos estados possiveis de um processo
const (
	ENTER dmxReq = iota
	EXIT
)

// Estrutura da mensagem de resposta do módulo DIMEX para a aplicação
type dmxResp struct {
}

type DIMEX_Module struct {
	Req       chan dmxReq  // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind       chan dmxResp // canal para informar aplicacao que pode acessar
	addresses []string     // endereco de todos, na mesma ordem
	id        int          // identificador do processo - é o indice no array de enderecos acima
	st        State        // estado deste processo na exclusao mutua distribuida
	waiting   []bool       // processos aguardando tem flag true
	lcl       int          // relogio logico local
	reqTs     int          // timestamp local da ultima requisicao deste processo
	nbrResps  int          // número de respostas recebidas
	dbg       bool
	Pp2plink *PP2PLink.PP2PLink // acesso aa comunicacao enviar por PP2PLinq.Req e receber por PP2PLinq.Ind
	Snapshots      map[int]*Snapshot
	MarkerReceived map[int]bool
	SnapshotMutex  sync.Mutex
	Recording      map[int]bool
	ChannelBuffers map[int][]string
}

type Snapshot struct {
	ID        int
	State     State
	ReqTs     int
	NbrResps  int
	Waiting   []bool
	ChannelStates map[int][]string
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {

	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)

	dmx := &DIMEX_Module{
		Req: make(chan dmxReq, 1),
		Ind: make(chan dmxResp, 1),

		addresses: _addresses,
		id:        _id,
		st:        noMX,
		waiting:   make([]bool, len(_addresses)),
		lcl:       0,
		reqTs:     0,
		nbrResps:  0,
		dbg:       _dbg,
		Pp2plink: p2p,
		Snapshots:      make(map[int]*Snapshot),
		MarkerReceived: make(map[int]bool),
		Recording:     make(map[int]bool),
		ChannelBuffers: make(map[int][]string),
	}

	for i := 0; i < len(dmx.waiting); i++ {
		dmx.waiting[i] = false
	}
	dmx.Start()
	dmx.outDbg("Init DIMEX!")
	return dmx
}

// ------------------------------------------------------------------------------------
// ------- nucleo do funcionamento
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) Start() {

	go func() {
		for {
			select {
			case dmxR := <-module.Req: // vindo da  aplicação
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry() // Solicita entrada na SC

				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit() // Solicita saída da SC
				}

			case msgOutro := <-module.Pp2plink.Ind: // Mensagem de outro processo
				if strings.Contains(msgOutro.Message, "respOK") {
					module.outDbg("Recebeu respOK! " + msgOutro.Message)
					module.handleUponDeliverRespOk(msgOutro)

				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					module.outDbg("Recebeu reqEntry " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro)
					
				} else if strings.Contains(msgOutro.Message, "marker") {
					module.outDbg("Recebeu snapshot " + msgOutro.Message)
					parts := strings.Split(msgOutro.Message, " ")
					snapshotID, _ := strconv.Atoi(parts[1])
					senderID, _ := strconv.Atoi(parts[2])
					module.handleMarkerMessage(snapshotID, senderID)
				}else {
					// Record message if we are recording the channel state
					for i := 0; i < len(module.addresses); i++ {
						if module.Recording[i] {
							module.ChannelBuffers[i] = append(module.ChannelBuffers[i], msgOutro.Message)
						}
					}
				}
			}
		}
	}()
}

// ------------------------------------------------------------------------------------
// ------- tratamento de pedidos vindos da aplicacao
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponReqEntry() {
	module.lcl++              // Incrementa o relógio lógico local
	module.reqTs = module.lcl  // Define o timestamp da requisição
	module.nbrResps = 0        // Reseta o contador de respostas
	module.st = wantMX         // Define o estado como "querendo SC"
	for i := 0; i < len(module.addresses); i++ {
		if module.id == i { // Não envia para si mesmo
			continue
		}
		// Envia a mensagem de requisição de entrada para todos os processos
		module.sendToLink(module.addresses[i], "reqEntry "+strconv.Itoa(module.id)+" "+strconv.Itoa(module.lcl), "")
	}
}

func (module *DIMEX_Module) handleUponReqExit() {
	module.st = noMX // Atualiza o estado para "fora da SC"
	// Envia respOK para os processos que estão esperando
	for i := 0; i < len(module.addresses); i++ {
		if module.waiting[i] {
			module.sendToLink(module.addresses[i], "respOK "+strconv.Itoa(module.id), "")
			module.waiting[i] = false
		}
	}
	module.outDbg("Liberou SC e notificou processos em espera")
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponDeliverRespOk(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	module.nbrResps++
	// Quando todas as respostas forem recebidas, pode entrar na SC
	if module.nbrResps == len(module.addresses)-1 {
		module.st = inMX
		module.Ind <- dmxResp{} // Informa a aplicação que pode acessar a SC
		module.outDbg("Pode acessar a SC")
	}
}

func (module *DIMEX_Module) handleUponDeliverReqEntry(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	parts := strings.Split(msgOutro.Message, " ")
	senderId, _ := strconv.Atoi(parts[1])
	senderTs, _ := strconv.Atoi(parts[2])

	// Verifica a prioridade para enviar respOK
	if module.st == noMX || (module.st == wantMX && before(senderId, senderTs, module.id, module.reqTs)) {
		module.sendToLink(module.addresses[senderId], "respOK "+strconv.Itoa(module.id), "")
	} else {
		module.waiting[senderId] = true
	}
}

// ------------------------------------------------------------------------------------
// ------- Funções Auxiliares
// ------------------------------------------------------------------------------------

func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId
	}
}

func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content,
	}
}

func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println("[DIMEX] " + s)
	}
}

func (snapshot *Snapshot) SaveToFile(processID int) error {
	filePath := fmt.Sprintf("./snapshots/snapshot_%d_process_%d.json", snapshot.ID, processID)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(snapshot)
	if err != nil {
		return err
	}

	return nil
}


func (module *DIMEX_Module) InitiateSnapshot(snapshotID int) {
	module.SnapshotMutex.Lock()
	defer module.SnapshotMutex.Unlock()

	// Record the local state
	snapshot := &Snapshot{
		ID:            snapshotID,
		State:         module.st,
		ReqTs:         module.reqTs,
		NbrResps:      module.nbrResps,
		Waiting:       append([]bool(nil), module.waiting...),
		ChannelStates: make(map[int][]string),
	}
	module.Snapshots[snapshotID] = snapshot

	// Save snapshot to file
	err := snapshot.SaveToFile(module.id)
	if err != nil {
		module.outDbg(fmt.Sprintf("Error saving snapshot: %v", err))
	}

	// Send marker messages to all other processes
	for i := 0; i < len(module.addresses); i++ {
		if i != module.id {
			module.sendToLink(module.addresses[i], fmt.Sprintf("marker %d %d", snapshotID, module.id), "")
		}
	}

	// Mark that this process has received a marker for this snapshot
	module.MarkerReceived[snapshotID] = true

	// Start recording the state of incoming channels
	for i := 0; i < len(module.addresses); i++ {
		if i != module.id {
			module.Recording[i] = true
			module.ChannelBuffers[i] = []string{}
		}
	}
}

func (module *DIMEX_Module) handleMarkerMessage(snapshotID int, senderID int) {
	module.SnapshotMutex.Lock()
	defer module.SnapshotMutex.Unlock()

	if !module.MarkerReceived[snapshotID] {
		// First marker received for this snapshot, record local state
		module.MarkerReceived[snapshotID] = true

		// Record the local state
		snapshot := &Snapshot{
			ID:            snapshotID,
			State:         module.st,
			ReqTs:         module.reqTs,
			NbrResps:      module.nbrResps,
			Waiting:       append([]bool(nil), module.waiting...),
			ChannelStates: make(map[int][]string),
		}
		module.Snapshots[snapshotID] = snapshot

		// Save snapshot to file
		err := snapshot.SaveToFile(module.id)
		if err != nil {
			module.outDbg(fmt.Sprintf("Error saving snapshot: %v", err))
		}

		// Send marker messages to all other processes
		for i := 0; i < len(module.addresses); i++ {
			if i != module.id {
				module.sendToLink(module.addresses[i], fmt.Sprintf("marker %d %d", snapshotID, module.id), "")
			}
		}
		// Start recording the state of incoming channels
		for i := 0; i < len(module.addresses); i++ {
			if i != module.id {
				module.Recording[i] = true
				module.ChannelBuffers[i] = []string{}
			}
		}
	} else {
		// Subsequent marker received, stop recording the state of the channel from senderID
		module.Recording[senderID] = false
		snapshot := module.Snapshots[snapshotID]
		if snapshot != nil {
			snapshot.ChannelStates[senderID] = append(snapshot.ChannelStates[senderID], "marker received")
		}
	}
}