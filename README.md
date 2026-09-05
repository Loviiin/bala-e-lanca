# okok-scale-logger

Worker em Go que escaneia a balança OKOK/Chipsea via BLE, calcula
composição corporal e grava no vault do Obsidian. Roda no Orange Pi Zero
3 como container Docker, atualizado automaticamente via GHCR + Watchtower.

Ver a estrutura de pacotes em `internal/` — cada um comentado com o que
faz e por quê.

---

## 1. Ambiente de desenvolvimento (Windows + Linux)

**Recomendação: WSL2 + Docker Desktop (backend WSL2) + VS Code.**

Por quê: você escreve o código uma vez, o Docker Desktop já builda pra
Linux/arm64 sem precisar de VM separada, e o terminal do WSL2 é onde você
roda `go test`, `go build`, `docker buildx`, etc — ambiente idêntico ao
que vai rodar no Orange Pi (que também é Linux).

### Passo a passo

1. **Instalar o WSL2**: no PowerShell como administrador, `wsl --install
   -d Ubuntu`. Reinicia quando pedir.
2. **Instalar o Docker Desktop** (docker.com) e nas configurações
   (Settings → Resources → WSL Integration) habilitar a integração com
   a distro Ubuntu que você instalou.
3. **Dentro do WSL2 (Ubuntu)**: instalar o Go —
   ```bash
   sudo apt update && sudo apt install -y golang-go
   # ou baixe uma versão mais nova direto de go.dev/dl se quiser
   ```
4. **VS Code**: instale a extensão "WSL" — abrindo uma pasta de dentro
   do terminal WSL com `code .`, o VS Code roda a parte de servidor
   dentro do Linux, então extensões de Go/Docker funcionam nativamente.
5. Clone esse projeto dentro do filesystem do WSL2 (não em `/mnt/c/...`
   — performance de I/O é bem pior atravessando o Windows).

### ⚠️ Sobre testar Bluetooth dentro do WSL2

O WSL2 **não enxerga o adaptador Bluetooth do Windows por padrão**. Duas
opções:

- **Mais simples (recomendada)**: teste a parte de scanning/decoding
  direto no **Orange Pi real** assim que tiver o binário compilado —
  ele é o ambiente Linux/BlueZ de produção mesmo, então testar lá já
  valida tudo de uma vez. É só `scp` o binário e rodar direto (sem
  Docker ainda) via SSH.
- **Alternativa**: usar o projeto `usbipd-win` pra passar um dongle BLE
  USB do Windows pro WSL2. Mais setup, só vale a pena se quiser testar
  Linux/BlueZ sem depender do Pi ligado.

Pra validar a lógica de **decodificação dos bytes** (sem precisar de
rádio nenhum), rode os testes unitários — eles usam os valores reais que
já capturamos:

```bash
go test ./internal/ble/... -v
go test ./internal/bia/... -v
```

---

## 2. Primeira configuração

```bash
git clone https://github.com/Loviiin/bala-e-lanca.git
cd bala-e-lanca

# gera o go.sum (precisa de internet liberada pro proxy do Go)
go mod tidy

cp config.example.yaml config.yaml
# edite config.yaml com altura/sexo/idade/peso esperado de cada pessoa
```

---

## 3. Loop de desenvolvimento

```bash
# compila pro Orange Pi (ARM64)
GOOS=linux GOARCH=arm64 go build -o worker ./cmd/worker

# manda pro Pi e roda direto (sem Docker), pra iterar rápido
scp worker orangepi@<ip-do-pi>:~/
ssh orangepi@<ip-do-pi> 'SCALE_MAC=A8:0B:6B:77:98:C7 CONFIG_PATH=~/config.yaml VAULT_DIR=~/vault-teste ./worker'
```

Só depois que isso estiver funcionando direitinho é que vale empacotar
no Docker — assim você separa "bug na minha lógica" de "bug no
Docker/BlueZ dentro do container".

---

## 4. Deploy automático (GHCR + Watchtower)

Fluxo, de ponta a ponta:

```
git push origin main
      │
      ▼
GitHub Actions (.github/workflows/build.yml)
  builda multi-arch (linux/arm64) via QEMU+buildx
      │
      ▼
push automático pra ghcr.io/Loviiin/bala-e-lanca:latest
      │
      ▼
Watchtower no Orange Pi (checando a cada 5 min)
  detecta imagem nova → pull → recria o container
```

### Setup único (uma vez só)

1. **No repositório GitHub**: nada a configurar — o workflow usa
   `secrets.GITHUB_TOKEN`, que já existe automaticamente.
2. **Tornar o pacote GHCR público** (mais simples) OU configurar login:
   - Público: vá em `github.com/users/Loviiin/packages` depois do
     primeiro push, ache o pacote, Package settings → Change visibility
     → Public. Assim nem o Orange Pi nem o Watchtower precisam de login.
   - Privado: no Orange Pi, `docker login ghcr.io` uma vez com um
     Personal Access Token (`read:packages`), e descomente a linha do
     `~/.docker/config.json` no `docker-compose.yml` pro Watchtower usar
     o mesmo login.
3. **No Orange Pi**, instale Docker + Docker Compose (uma vez):
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER
   # relogue pra aplicar o grupo
   ```
4. Copie `docker-compose.yml` e `config.yaml` pro Pi, ajuste o caminho
   do volume do vault e o `SCALE_MAC`, e suba:
   ```bash
   docker compose up -d
   ```

Daqui pra frente, todo `git push` na main atualiza o Pi sozinho em até 5
minutos, sem você tocar em nada lá.

---

## 5. Riscos conhecidos (vale testar cedo, não no fim)

- **BLE dentro do container**: `network_mode: host` + montar
  `/var/run/dbus` costuma resolver, mas teste isso especificamente antes
  de confiar no pipeline inteiro — é o ponto mais frágil do projeto.
- **Permissão de escrita no vault**: a imagem final usa o usuário
  `nonroot` do distroless (UID 65532). Se o volume do vault no host tiver
  outro dono, o container não vai conseguir escrever. Ajuste com
  `chown -R 65532:65532 /home/orangepi/obsidian-vault/Saude` ou troque
  a base image do Dockerfile pra uma variante root se preferir simplicidade.
- **go.sum**: os arquivos desse esqueleto não incluem um `go.sum` válido
  (esse ambiente de setup não tinha acesso ao proxy do Go). Rode
  `go mod tidy` na sua máquina antes do primeiro build.
