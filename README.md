APRESENTAÇÃO:  30.set  , horário de aula, LABs 409/412

Crie uma pasta cujo nome é
    a concatenação dos nomes completos dos participantes, 
    separados por "_"
    em ordem alfabética
           Ex.:   BarbaraLiskov_EdmundClarke_LeslieLamport
Coloque seus fontes nesta pasta.
Comprima a pasta com    .ZIP                        (NAO use RAR)
Envie o .ZIP   na sala de entrega

Atenção: grupos de até 4.

ENUNCIADO

Parte 1)  Exclusão mútua distribuída.
Implemente o algoritmo de exclusão mútua distribuída visto em aula (Ricart/Agrawalla).
Os slides da disciplina mostram o algoritmo.    Um template em go foi provido pelo professor (vide módulo 1).
Você pode implementar em qualquer linguagem, desde que use um design reativo, como explicado em aula.
A aplicação fornecida, que acessa um arquivo compartilhado por diferentes processos,
deve rodar sem erros, conforme discutimos em aula.
Parte 2) Implemente o algoritmo de snapshot junto ao DiMEx, usando o algoritmo de Chandy-Lamport discutido em aula.

Note que na exclusão mútua, todo processo tem um canal com cada outro
e que este canal preserva ordem e não perde mensagens.   Assim, as suposições
do algoritmo de Chandy-Lamport são satisfeitas.
Conforme o algoritmo, o módulo DiMEx, estendido para snapshot, pode receber/tratar também uma mensagem de snapshot.
Cada snapshot tem um identificador único criado no processo que inicia o mesmo.   
Todo processo, ao gravar seu estado, grava este identificador junto.
O estado deve incluir suas variáveis e o estado dos seus canais de entrada, conforme o algoritmo de snapshot.
Os (diversos) snapshots completos devem ser avaliados junto ao funcionamento do sistema.

Realize as seguintes etapas, e demonstre os resultados no dia da apresentação.

0) rode o DIMEX com no mínimo 3 processos.   

1) faça um processo iniciar snapshots sucessivos, cada um com um identificador (1, 2, 3 ...)
    concorrentemente aos seus acessos como um processo usuário do DIMEX
2) colha uma sequencia de snapshots.   algumas centenas.
     eles devem estar em arquivos separados, um para cada processo
3) escreva uma ferramenta que avalia para cada snapshot se os estados dos processos estão consistentes.
    Para cada snapshot SnId a ferramenta lê os estados gravados por cada processo, respectivo ao snapshot SnId,
    e avalia se o mesmo está correto.
    Para isso voce tem que enunciar invariantes do sistema.   Invariante é algo que deve ser verdade em qualquer estado.
    Exemplos
      Inv  1:   no máximo um processo na SC.
      inv  2:  se todos processos estão em "não quero a SC", então todos waitings tem que ser falsos e não deve haver mensagens
      inv 3:   se um processo q está marcado como waiting em p, então p está na SC ou quer a SC
      inv 4:   se um processo q quer a seção crítica (nao entrou ainda),
                 então o somatório de mensagens recebidas, de mensagens em transito e de, flags waiting para p em outros processos
                 deve ser igual a N-1  (onde N é o número total de processos)
      inv ... etc.

      Cada invariante é um teste sobre um snapshot, uma   funcao_InvX(snapshot)      retorna um bool com o resultado
      Cada snapshot é avaliado para todas invariantes.
      A ferramenta avisa invariantes violadas e o snapshot.

3) rode o sistema e avalie com a ferramenta - 
      se ela gerou avisos de violacao de invariantes,  avalie seu algoritmo (ou o algoritmo de snapshot)
      para o DIMEX supostamente correto, as invariantes devem todas passar.

4) insira falhas no DIMEX
      por exemplo, altere a condicao de resposta para violar a SC,
                               altere a mesma condicao para bloquear 

5)  detecte estes casos com a análise de snapshots
