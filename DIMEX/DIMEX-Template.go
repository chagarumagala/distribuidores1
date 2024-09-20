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
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type State int
const (
	noMX State = iota
	wantMX
	inMX
)

type dmxReq int
const (
	ENTER dmxReq = iota
	EXIT
)

// Declaração de dmxResp corrigida
type dmxResp struct {
	// Pode estar vazia; usada apenas como um sinal de que o processo pode acessar a SC
}

type SnapshotMessage struct {
	SnapshotId int
}

type SnapshotState struct {
	ProcessState State
	Waiting      []bool
	Timestamp    int
	Messages     []string
}

type DIMEX_Module struct {
	Req          chan dmxReq      // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind          chan dmxResp     // canal para informar aplicacao que pode acessar
	SnapshotReq  chan SnapshotMessage // canal para solicitar snapshots
	addresses    []string         // endereco de todos os processos
	id           int              // identificador do processo
	st           State            // estado do processo
	waiting      []bool           // processos aguardando tem flag true
	lcl          int              // relogio logico local
	reqTs        int              // timestamp local da ultima requisicao
	nbrResps     int              // contador de respostas
	snapshots    map[int]SnapshotState // estados de snapshots capturados
	snapshotLock sync.Mutex       // mutex para proteger estados de snapshot
	dbg          bool
	Pp2plink     *PP2PLink.PP2PLink   // canal de comunicação com outros processos
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {
	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)
	dmx := &DIMEX_Module{
		Req:          make(chan dmxReq, 1),
		Ind:          make(chan dmxResp, 1),
		SnapshotReq:  make(chan SnapshotMessage, 1),
		addresses:    _addresses,
		id:           _id,
		st:           noMX,
		waiting:      make([]bool, len(_addresses)),
		lcl:          0,
		reqTs:        0,
		nbrResps:     0,
		snapshots:    make(map[int]SnapshotState),
		dbg:          _dbg,
		Pp2plink:     p2p,
	}

	for i := 0; i < len(dmx.waiting); i++ {
		dmx.waiting[i] = false
	}
	dmx.Start()
	dmx.outDbg("Init DIMEX!")
	return dmx
}

// ------------------------------------------------------------------------------------
// ------- núcleo do funcionamento
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) Start() {
	go func() {
		for {
			select {
			case dmxR := <-module.Req: // Pedidos da aplicação
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry()
				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit()
				}
			case snapshotReq := <-module.SnapshotReq: // Pedidos de snapshot
				module.handleUponSnapshot(snapshotReq.SnapshotId)
			case msgOutro := <-module.Pp2plink.Ind: // Mensagens de outros processos
				if strings.Contains(msgOutro.Message, "respOK") {
					module.outDbg("Recebeu respOK! " + msgOutro.Message)
					module.handleUponDeliverRespOk(msgOutro)
				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					module.outDbg("Recebeu reqEntry " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro)
				} else if strings.Contains(msgOutro.Message, "snapshot") {
					module.processSnapshot(msgOutro)
				}
			}
		}
	}()
}

// ------------------------------------------------------------------------------------
// ------- tratamento de snapshots
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponSnapshot(snapshotId int) {
	// Captura o estado local
	module.snapshotLock.Lock()
	defer module.snapshotLock.Unlock()

	snapshot := SnapshotState{
		ProcessState: module.st,
		Waiting:      append([]bool{}, module.waiting...),
		Timestamp:    module.lcl,
		Messages:     []string{},
	}

	// Armazena o snapshot
	module.snapshots[snapshotId] = snapshot

	// Envia mensagem de snapshot para todos os outros processos
	for i := 0; i < len(module.addresses); i++ {
		if module.id != i {
			module.sendToLink(module.addresses[i], "snapshot "+strconv.Itoa(snapshotId), "")
		}
	}
	module.outDbg(fmt.Sprintf("Iniciou snapshot %d", snapshotId))
}

func (module *DIMEX_Module) processSnapshot(msg PP2PLink.PP2PLink_Ind_Message) {
	// Parse da mensagem de snapshot
	parts := strings.Split(msg.Message, " ")
	snapshotId, _ := strconv.Atoi(parts[1])

	// Se o snapshot ainda não foi capturado, captura o estado local
	module.snapshotLock.Lock()
	defer module.snapshotLock.Unlock()

	if _, exists := module.snapshots[snapshotId]; !exists {
		snapshot := SnapshotState{
			ProcessState: module.st,
			Waiting:      append([]bool{}, module.waiting...),
			Timestamp:    module.lcl,
			Messages:     []string{}, // aqui estariam mensagens em trânsito
		}
		module.snapshots[snapshotId] = snapshot
		module.outDbg(fmt.Sprintf("Processou snapshot %d", snapshotId))
	}
}

// ------------------------------------------------------------------------------------
// ------- tratamento de pedidos vindos da aplicacao
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponReqEntry() {
	module.lcl++
	module.reqTs = module.lcl
	module.nbrResps = 0
	module.st = wantMX
	for i := 0; i < len(module.addresses); i++ {
		if module.id != i {
			module.sendToLink(module.addresses[i], "reqEntry "+strconv.Itoa(module.id)+" "+strconv.Itoa(module.lcl), "")
		}
	}
}

func (module *DIMEX_Module) handleUponReqExit() {
	module.st = noMX
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
	if module.nbrResps == len(module.addresses)-1 {
		module.st = inMX
		module.Ind <- dmxResp{}
		module.outDbg("Pode acessar a SC")
	}
}

func (module *DIMEX_Module) handleUponDeliverReqEntry(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	parts := strings.Split(msgOutro.Message, " ")
	senderId, _ := strconv.Atoi(parts[1])
	senderTs, _ := strconv.Atoi(parts[2])
	if module.st == noMX || (module.st == wantMX && before(senderId, senderTs, module.id, module.reqTs)) {
		module.sendToLink(module.addresses[senderId], "respOK "+strconv.Itoa(module.id), "")
	} else {
		module.waiting[senderId] = true
	}
}

// ------- Funções Auxiliares -------

// Função auxiliar para comparar timestamps e definir a prioridade entre processos
func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId // Desempate usando o ID do processo
	}
}

// Função auxiliar para enviar uma mensagem via PP2PLink
func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
	module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + content)
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content,
	}
}

// Função auxiliar para exibir mensagens de debug
func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println("[DIMEX] " + s)
	}
}