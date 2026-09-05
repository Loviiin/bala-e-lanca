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

### Setup no Orange Pi

Execute os comandos abaixo no Orange Pi via SSH. O exemplo assume o
usuário `orangepi` e um Orange Pi Zero 3 com Linux ARM64.

#### 1. Conferir o sistema e instalar dependências

```bash
uname -m                         # esperado: aarch64
sudo apt update
sudo apt install -y bluez dbus curl ca-certificates
sudo systemctl enable --now bluetooth
```

Instale Docker e habilite o uso sem `sudo`:

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker "$USER"
```

Saia e entre novamente no SSH para o grupo Docker ser aplicado. Confirme:

```bash
docker --version
docker compose version
```

#### 2. Preparar o vault e a configuração

```bash
mkdir -p ~/okok-scale-logger
mkdir -p ~/obsidian-vault/Saude
cd ~/okok-scale-logger
```

Copie `docker-compose.yml` e `config.yaml` do seu computador para essa
pasta. No `docker-compose.yml`, confira:

- `image: ghcr.io/Loviiin/bala-e-lanca:latest`;
- o MAC em `SCALE_MAC`;
- o caminho do vault em `/home/orangepi/obsidian-vault/Saude`.

O `config.yaml` não deve ser commitado no GitHub, pois contém os dados dos
perfis. Use `config.example.yaml` como modelo.

#### 3. Testar o Bluetooth no host

Antes do container, confirme que o BlueZ enxerga a balança:

```bash
sudo bluetoothctl
power on
scan on
```

Deixe a balança transmitindo e procure o MAC `A8:0B:6B:77:98:C7`. Para
sair do `bluetoothctl`, execute `scan off` e `quit`.

#### 4. Acessar o GHCR

Se o pacote `ghcr.io/Loviiin/bala-e-lanca` for público, não precisa fazer
login. Se for privado, crie um GitHub Personal Access Token com permissão
`read:packages` e execute no Orange Pi:

```bash
docker login ghcr.io -u Loviiin
```

Use o token como senha. Para o Watchtower reutilizar esse login, descomente
no `docker-compose.yml`:

```yaml
- ~/.docker/config.json:/config.json:ro
```

#### 5. Subir e acompanhar o serviço

Crie o arquivo local de credenciais do bridge (ele não deve ser commitado):

```bash
cp .env.livesync.example .env.livesync
nano .env.livesync
```

Preencha `COUCHDB_PASSWORD` com a senha atual do CouchDB. O bridge observa
`/home/loviin/obsidian-vault` e envia os Markdown para a base
`obsidian-livesync`, sem precisar rodar o Obsidian no Orange Pi.

```bash
cd ~/okok-scale-logger
docker compose pull
docker compose up -d
docker compose ps
docker compose logs -f okok-scale-logger
docker compose logs -f obsidian-livesync-bridge
```

Quando uma pesagem estável for detectada, o log deve mostrar o peso e a
gravação no vault. O Watchtower verifica uma nova imagem a cada 5 minutos.

#### 6. Diagnóstico rápido

```bash
docker compose logs --tail=100 okok-scale-logger
docker inspect okok-scale-logger
systemctl status bluetooth --no-pager
ls -l /var/run/dbus
```

Se o container não conseguir escrever no vault, ajuste a permissão para o
usuário `nonroot` da imagem:

```bash
sudo chown -R 65532:65532 ~/obsidian-vault/Saude
```

Depois de qualquer atualização publicada pelo GitHub Actions, é possível
forçar a atualização manualmente:

```bash
docker compose pull
docker compose up -d
```

Normalmente o Watchtower faz isso sozinho em até 5 minutos.

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
