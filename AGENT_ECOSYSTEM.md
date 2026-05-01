# Ecossistema Multi-Agente do `jitctx`

> **Pipeline de implementação de features de ponta a ponta**, orientado por requisitos
> formais (US-/RF-/RNF-XXX) e materializado em código Go que respeita rigidamente
> a arquitetura DDD/Clean do projeto.

Este documento descreve **como os agentes se comunicam, como o paralelismo é
extraído, qual o papel das skills e por que os guidelines são lei** — ou seja,
todo o ecossistema que torna o pipeline efetivo.

---

## 1. Visão geral em 30 segundos

```
requirements.md / .feature
        │
        ▼
┌──────────────────────┐
│  @agent-manager      │  Team Lead — orquestra, NUNCA escreve código
└─────────┬────────────┘
          │ Step 1
          ▼
┌──────────────────────┐    plan.md (sections 0-9)
│ @discovery-planning  │ ─────────────────────────► .claude/plans/{feature}/plan.md
└─────────┬────────────┘
          │ Step 2 — PAUSA HUMANA (única no pipeline)
          ▼
   Aprovação do plano
          │ Step 3 — Tier walk
          ▼
┌──────────────────────────────────────────────┐
│  @implementation × N  (paralelos por tier)   │  uma sessão por GROUP_ID
└─────────┬────────────────────────────────────┘
          │ Step 4 — git diff + união dos summaries
          ▼
┌──────────────────────┐
│  @qa-coordinator     │  ←─ Step 5
└────┬───────────┬─────┘
     │ paralelo  │ paralelo
     ▼           ▼
┌──────────┐ ┌────────────┐
│ @sec-gate│ │ @code-rev  │
└────┬─────┘ └─────┬──────┘
     │ reports     │ reports
     ▼             ▼
.claude/reports/{security,code-review}-{feature}.md
          │
          │ até 3 ciclos de QA_FIX (uma sessão @implementation por ciclo)
          ▼
┌──────────────────────┐
│  @commit-helper      │  ←─ Step 6 — single Conventional Commit
└─────────┬────────────┘
          ▼
   Step 7 — Final Summary
```

**Resultado autônomo:** após a aprovação do plano (Step 2), o pipeline corre
sozinho até o commit. QA nunca bloqueia — findings não resolvidos viram
deferred items no resumo final.

---

## 2. Catálogo de agentes

Cada agente vive em `.claude/agents/<nome>.md` e declara, no frontmatter,
**quais ferramentas tem acesso** e **qual skill carrega**. O frontmatter é o
contrato técnico do agente.

| Agente | Modelo | Papel | Pode delegar para | Escreve código? | Escreve onde? |
|---|---|---|---|---|---|
| `agent-manager` | opus-4-7 | Team Lead / orquestrador | discovery-planning, implementation, qa-coordinator, commit-helper | ❌ | nenhum lugar |
| `discovery-planning` | opus-4-7 (effort: xhigh) | Arquiteto — produz `plan.md` | — | ❌ | `.claude/plans/{feature}/` |
| `implementation` | sonnet-4-6 | Engenheiro — Go puro | — | ✅ | `internal/`, `cmd/`, `testdata/` |
| `qa-coordinator` | opus-4-7 | QA Lead | security-gate, code-reviewer, implementation (QA_FIX) | ❌ | nenhum lugar |
| `security-gate` | opus-4-7 (high) | Auditor de segurança (read-only) | — | ❌ | `.claude/reports/` |
| `code-reviewer` | sonnet-4-6 (high) | Revisor de qualidade (read-only) | — | ❌ | `.claude/reports/` |
| `commit-helper` | haiku-4-5 | Empacotador git | — | ❌ | apenas commit |

### Por que essa segregação importa

- **Princípio da responsabilidade única em escala de agente.** O Manager não
  escreve, o Implementer não revisa, o Reviewer não corrige. Cada papel tem
  uma única forma de "errar", e quando algo dá errado é trivial saber qual
  loop falhou.
- **Read-only é literal.** `security-gate` e `code-reviewer` só escrevem em
  `.claude/reports/`. Qualquer correção passa obrigatoriamente pelo
  `@implementation QA_FIX` — assim toda alteração de código vai pelo mesmo
  pipeline de testes (`gofmt`/`go vet`/`go test`).

---

## 3. Model tier + effort — alocação consciente de recurso

Os campos `model:` e `effort:` no frontmatter de cada agente **não são
decoração**. Eles definem dois eixos ortogonais:

- **`model`** — o **teto cognitivo** do agente. Opus raciocina mais
  profundamente, custa mais por token, é mais lento; Sonnet é o equilíbrio;
  Haiku é rápido e barato para tarefas mecânicas.
- **`effort`** — o **orçamento de "thinking"** que o agente pode gastar
  antes de responder. Quanto maior o effort, mais o modelo pode iterar
  internamente, considerar alternativas, validar self-checks. Custa tokens
  e latência diretamente proporcional.

A combinação `model × effort` é o **dial de custo/qualidade por papel**.
Agentes que erram caro (planejamento, auditoria de segurança) recebem
combinações generosas; agentes que executam decisões já tomadas recebem
combinações enxutas.

### Matriz atual

| Agente | Model | Effort | Custo relativo | Por que essa combinação |
|---|---|---|---|---|
| `agent-manager` | **opus-4-7** | medium | ~~~ | Orquestração: sem código, mas precisa parsear YAML do Section 9, reconciliar Phase 10 summaries de N teammates paralelos, aplicar Failure Policy (FAILED/SKIPPED/SUCCESS) e decidir quando re-rodar QA. Erro aqui contamina todo o pipeline. Effort medium — não está raciocinando sobre código, está roteando. |
| `discovery-planning` | **opus-4-7** | **xhigh** | ~~~~~ | É o agente mais caro do pipeline e isso é proposital. Ele lê requisitos ambíguos, scaneia o codebase, congela um Domain Contract que **N teammates downstream vão consumir como verdade**. Erro de discovery é amplificado por todos os tiers. xhigh dá o budget de raciocínio para evitar `CONTRACT_MISMATCH` na Tier 2+. Roda **uma vez por feature** — investe-se pesado em uma única chamada para baratear tudo depois. |
| `implementation` | **sonnet-4-6** | medium | ~~ | Implementação a partir de um plano detalhado é, por construção, **mecânica**: o plano já listou arquivos, contratos, guidelines, fases. Sonnet é mais que suficiente. Effort medium porque os test gates (`gofmt`/`vet`/`test`) servem de feedback loop — se o código está errado, ele sabe via teste, não via thinking interno. Roda **N vezes em paralelo** por tier — pagar Opus aqui multiplicaria o custo sem ganho mensurável. |
| `qa-coordinator` | **opus-4-7** | medium | ~~~ | Decide se um finding é fixável, monta SCOPE da união de dois reports heterogêneos, controla o loop de até 3 ciclos. Não escreve código nem markdown — apenas roteamento de decisão. Opus pelo mesmo motivo do agent-manager (decisões com consequência sistêmica), effort medium pelo mesmo motivo (raciocina sobre paths, não sobre Go). |
| `security-gate` | **opus-4-7** | **high** | ~~~~ | Auditoria de segurança é exatamente o tipo de tarefa que **modelos menores erram silenciosamente**: alucinam CVEs, perdem path traversal sutis, ignoram subprocess injection. Opus + high effort é a aposta de "prefiro pagar por análise profunda agora do que descobrir um CVE em produção". Roda só ao final, no loop de QA — custo concentrado, não distribuído. |
| `code-reviewer` | **sonnet-4-6** | **high** | ~~~ | Revisão é estruturada (4 dimensões, traceback obrigatório para guideline). Sonnet entrega qualidade comparável a Opus para checagens deterministas contra regras escritas, e é ~3-5× mais barato. Effort high porque a tarefa exige varrer todo SCOPE com múltiplos critérios — cortar effort aqui vira BLOCKER faltante. |
| `commit-helper` | **haiku-4-5** | medium | ~ | A diff já existe, os requirement IDs já estão no plan.md Section 0. Tudo que falta é gerar uma string Conventional Commits e chamar `git commit`. **Tarefa puramente mecânica** — Haiku é a escolha óbvia. Effort medium porque o agente ainda precisa agrupar arquivos por concern lógico para o body do commit. |

### Os trade-offs explícitos

1. **Onde a falha sai cara, paga-se Opus.** Plan errado contamina seis
   teammates downstream; CVE não detectado vai para produção; orquestração
   confusa quebra o pipeline inteiro. Esses três agentes (manager,
   discovery, qa-coordinator, security-gate) são **caros por design**.

2. **Onde a falha é detectável e corrigível, paga-se Sonnet/Haiku.**
   Implementation tem `gofmt`/`vet`/`test` como gate. Code-reviewer tem
   guidelines escritos como ground-truth. Commit-helper tem o git como
   verificação trivial. Modelos menores aqui entregam o mesmo resultado.

3. **Effort escala com a profundidade do raciocínio, não com a importância
   do agente.** O `agent-manager` é crítico mas roda effort `medium` —
   porque ele **decide rotas**, não **raciocina sobre código**. Já o
   `discovery-planning` (xhigh) e `security-gate` (high) precisam de
   thinking budget alto porque a saída deles é uma análise, não uma
   roteamento.

4. **Concentração temporal importa.** Discovery roda **1×** por feature.
   Implementation roda **N×** em paralelo. Logo, é racional gastar Opus +
   xhigh em discovery (1 chamada cara) e Sonnet + medium em implementation
   (N chamadas baratas). Inverter isso multiplicaria o custo sem ganho.

### Como mudar a matriz

Os campos vivem no frontmatter de `.claude/agents/<nome>.md`. Mudar um
agente é trivial — não há código de orquestração que dependa do model:

```yaml
---
name: implementation
model: claude-sonnet-4-6   # ← troque aqui se precisar
effort: medium             # ← ou aqui
---
```

Sinais de que um ajuste é necessário:

- **`code-reviewer` deixando passar BLOCKERs óbvios** → subir para Opus
  ou subir effort para `xhigh`.
- **`discovery-planning` produzindo `Open Questions: blocking` recorrentes
  em features simples** → effort `xhigh` está virando overthinking; testar
  `high`.
- **`implementation` com taxa de retry alta nos test gates** → ou o plano
  está vago (problema upstream em discovery), ou Sonnet não está dando
  conta — escalar para Opus localmente, não globalmente.
- **Latência da feature dominada por discovery** → aceitar; é o trade-off
  consciente. Cortar discovery quase sempre piora downstream.

---

## 4. Comunicação entre agentes

Não há barramento de eventos nem fila de mensagens — a comunicação é
**hierárquica, por chamada de teammate**, com **payload restrito a paths**.

### 4.1 Padrão "passe paths, nunca conteúdo"

Cada agente carrega o que precisa do disco. Os prompts de delegação ficam
abaixo de ~500 tokens. Nada de empurrar o relatório inteiro para o filho.

Exemplo concreto, do `qa-coordinator` para `@implementation` em modo QA_FIX:

```
MODE:                 QA_FIX
SECURITY_REPORT_PATH: .claude/reports/security-report-{feature}.md
REVIEW_REPORT_PATH:   .claude/reports/code-review-{feature}.md
SCOPE:                {união dos arquivos flagados}
```

### 4.2 Artefatos como contratos

Os agentes não compartilham estado em memória — só **arquivos versionáveis**:

| Artefato | Produtor | Consumidor | Função |
|---|---|---|---|
| `.claude/plans/{feature}/plan.md` | discovery-planning | agent-manager, implementation | Plano completo + Frozen Domain Contract + YAML do Section 9 |
| `.claude/reports/security-report-{feature}.md` | security-gate | qa-coordinator, implementation | Findings de segurança + fix instructions |
| `.claude/reports/code-review-{feature}.md` | code-reviewer | qa-coordinator, implementation | BLOCKERs/WARNINGs/INFOs |
| Phase 10/15 summary (texto) | implementation | agent-manager / qa-coordinator | Files created/modified + test results |
| `git diff --name-only HEAD` | git | agent-manager | Authoritative scope para QA |

Isso permite **inspeção humana em qualquer ponto do pipeline** — todos os
artefatos são markdown ou diff, nada é black-box.

### 4.3 Hierarquia de delegação

```
agent-manager
    ├── discovery-planning           (1 chamada)
    ├── implementation × N           (1 por GROUP_ID, em paralelo dentro do tier)
    ├── qa-coordinator               (1 chamada — ele orquestra o loop)
    │      ├── security-gate         (paralelo)
    │      ├── code-reviewer         (paralelo)
    │      └── implementation QA_FIX (1 sessão por ciclo, máx 3 ciclos)
    └── commit-helper                (1 chamada)
```

**Regra dura:** `security-gate`, `code-reviewer` e `commit-helper` **não
delegam para ninguém**. O fan-out de implementation está concentrado no
`agent-manager` e o de QA_FIX no `qa-coordinator`.

---

## 5. Paralelismo — extraído do DAG, não imposto

`jitctx` é um único codebase Go (sem split front/back, sem servidor, sem
banco). O paralelismo vem de **independência em nível de arquivo dentro de
uma camada DDD**, e é codificado no plano como um grafo dirigido acíclico de
**tiers** com **groups**.

### 5.1 Mapa de Tiers (definido em `discovery/SKILL.md`)

| Tier | Conteúdo | Depende de | Granularidade dos groups |
|---|---|---|---|
| **1** | `internal/domain/**` | ∅ | sempre 1 group (consistência interna do domínio) |
| **2** | `internal/infrastructure/{collaborator}/**` | [1] | 1 group por collaborator (`fsmanifest`, `fsprofile`, `treesitter`, `token`) |
| **3** | `internal/application/usecase/{action}uc/**` | [1] | 1 group por use case |
| **4** | `internal/cli/command/*Cmd.go`, `internal/cli/format/*.go` | [1, 3] | 1 group por command + 1 para formatters |
| **5** | `wire.go`, `root.go`, `execute.go`, `cmd/jitctx/main.go`, `internal/config/**` | [2, 3, 4] | sempre 1 group (composição é ponto único) |
| **6** | testes unitários e de integração | [5] | 1 group por arquivo de teste |

### 5.2 Section 9 do plan.md — o YAML autoritativo

Esse é **o único bloco que o `agent-manager` parseia para spawnar trabalho**.
Cada group declara `id`, `scope.create`/`scope.modify`, `guidelines`,
`effort`, `notes`. O Manager extrai por regex e roda `yaml.Unmarshal`.

```yaml
tiers:
  - id: 1
    name: Domain contract
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create: [internal/domain/port/parser/parseJavaFilePort.go, ...]
          modify: [internal/domain/errors/errors.go]
        guidelines: [.claude/guidelines/domain-layer-guidelines.yml]
        effort: M

  - id: 2
    name: Infrastructure adapters (parallel)
    depends_on: [1]
    groups:
      - id: T2-G1   # fsmanifest
      - id: T2-G2   # fsprofile
      - id: T2-G3   # treesitter
      - id: T2-G4   # token
    # ... os 4 rodam em paralelo, esperando apenas Tier 1
```

### 5.3 Algoritmo do tier-walk (no `agent-manager`)

```
for t in tiers ordenados por id:
    se algum dep ∉ SUCCESS → marca t como SKIPPED, continua
    spawn @implementation em paralelo, um por group em t.groups
    aguarda TODOS terminarem (barreira)
    se algum FAILED → t = FAILED (mas o pipeline avança)
    senão → t = SUCCESS
```

**Invariantes que sustentam o paralelismo:**

1. **Cada arquivo aparece em exatamente um group em todo o plano.** Sem
   merge conflicts entre teammates.
2. **`scope.create + scope.modify` é a fronteira de escrita do teammate.**
   Tocar fora disso é `PLANNER_ERROR` — o teammate aborta, não estende
   silenciosamente.
3. **Section 2 (Frozen Domain Contract) é read-only para tiers ≥ 2.** Se o
   teammate detecta que cumprir o trabalho violaria o contrato, ele retorna
   `CONTRACT_MISMATCH` e o pipeline volta à discovery. Não se patcha em
   torno do contrato.
4. **Falha intra-tier não mata os outros groups do mesmo tier.** Falha
   inter-tier propaga `SKIPPED` para tiers que dependem do FAILED. QA roda
   na feature parcial assim mesmo — verdade > teatro de sucesso.

---

## 6. Skills — onde mora a competência

Agentes são **finos** (frontmatter + ~30 linhas de prompt). A inteligência de
domínio está nas **skills** em `.claude/skills/<nome>/SKILL.md`. Cada agente
declara `skills: [nome]` no frontmatter e a skill é injetada no prompt em
runtime.

| Skill | Carregada por | O que faz |
|---|---|---|
| `discovery` | discovery-planning | Lê requisitos + scaneia o codebase + gera `plan.md` (Section 0-9) com Frozen Domain Contract |
| `implementation` | implementation | Traduz plan.md em Go por camada (Phase 2-9), modos Standard e Fix |
| `security-gate` | security-gate | Audita 4 pilares (CVEs, filesystem, secrets, config) — read-only |
| `pr-review` | code-reviewer | Revisão em 4 dimensões (arquitetura, idioms, code-smell, test-consistency) |
| `commit-helper` | commit-helper | Conventional Commits referenciando US-/RF- |
| `requirements-elicitation` | (standalone) | Elicitação domain-agnostic — Gherkin, MoSCoW, RICE, etc. |

### Por que separar agente de skill

- **Reuso vertical.** A mesma skill `implementation` serve Standard Mode
  (full plan) e Fix Mode (QA_FIX/SECURITY_FIX/CODE_REVIEW_FIX) — três
  contextos, um corpo de conhecimento.
- **Trocar o modelo do agente sem reescrever o método.** Mudei o
  `code-reviewer` de Opus para Sonnet sem tocar em `pr-review/SKILL.md`.
- **`requirements-elicitation` é standalone** — não conhece DDD, não conhece
  Go. É portátil; pode rodar em qualquer projeto. Fica isolada da tubulação
  específica do `jitctx` por design.

---

## 7. Guidelines — a constituição do projeto

Os arquivos em `.claude/guidelines/*.yml` são a **fonte única de verdade
para "como código deste projeto deve parecer"**. São referenciados pelos
plans, pelos teammates, pelas skills de QA.

| Guideline | Quem consome | Quando carrega |
|---|---|---|
| `domain-layer-guidelines.yml` | discovery, implementation (Phase 2), pr-review | sempre que algo em `internal/domain/**` muda |
| `application-layer-guidelines.yml` | discovery, implementation (Phase 3), pr-review | sempre que `internal/application/**` muda |
| `infrastructure-layer-guidelines.yml` | discovery, implementation (Phase 4), pr-review, security-gate | adapters em `internal/infrastructure/**` |
| `presentation-layer-guidelines.yml` | discovery, implementation (Phase 5), pr-review | cobra commands e formatters |
| `main-layer-guidelines.yml` | discovery, implementation (Phase 6), pr-review | composition root + config |
| `unit-test-layer-guidelines.yml` | discovery, implementation (Phase 7), pr-review | qualquer `*_test.go` não-integração |
| `integration-test-layer-guidelines.yml` | discovery, implementation (Phase 8), pr-review | `*Integration_test.go` |

### Como os guidelines são "lei"

1. **Discovery referencia guidelines no Section 9 do plano.** Cada `group`
   declara explicitamente os guidelines que sua implementação deve carregar.
   Se o group não escreve em uma camada, o guideline daquela camada **não é
   carregado** — economia de contexto.

2. **Implementation só carrega o guideline da fase que está executando.**
   Phase 2 carrega o domain. Phase 4, o de infrastructure. Não se polui o
   contexto do teammate com regras que ele não aplica.

3. **Code review faz traceback de cada BLOCKER para uma seção de
   guideline.** Não há regra "porque sim" — toda violação aponta para
   `domain-layer-guidelines.yml#port` ou similar.

4. **YAML é validado.** Há um script em `scripts/main.go` que parseia todos
   os YAMLs e falha o build se algum for inválido. Editar guidelines obriga
   `cd scripts && go run main.go ../.claude/guidelines/*.yml`.

5. **Guidelines não são modificados pelo pipeline.** Apenas o humano edita
   `.claude/guidelines/*.yml`. Plan, reports, summaries — tudo isso é
   regenerável; guidelines são durabilidade.

---

## 8. Memória, planos e relatórios — três persistências distintas

| Tipo | Diretório | Lifetime | Quem escreve | Para quê |
|---|---|---|---|---|
| **Guidelines** | `.claude/guidelines/` | permanente, versionado | só humano | constituição |
| **Plans** | `.claude/plans/{feature}/` | uma execução | discovery-planning | contrato da feature |
| **Reports** | `.claude/reports/` | uma execução, mas histórico mantido | security-gate, code-reviewer | auditoria |
| **Memory (Claude)** | `~/.claude/.../memory/` | inter-conversa | Claude (auto) | preferências do usuário, contexto de projeto |
| **Source code** | `internal/`, `cmd/` | permanente, versionado | implementation | a feature em si |

**Regras de escrita rígidas (não-negociáveis):**

- Discovery NUNCA escreve em `internal/`, `cmd/`, ou `.claude/reports/`.
- Implementation NUNCA escreve em `.claude/plans/` ou `.claude/reports/`.
- QA agents (security-gate, code-reviewer) escrevem APENAS em
  `.claude/reports/`.
- agent-manager e qa-coordinator NUNCA escrevem código nem markdown — eles
  apenas orquestram.
- commit-helper NUNCA modifica source — apenas chama git.

Esse cross-isolation impede classes inteiras de bug por agentes que
extrapolam papel.

---

## 9. Modos do `@implementation`

O mesmo agente atende dois ciclos distintos do pipeline. A skill define
ambos — o frontmatter do payload diferencia.

### Standard Mode (build do feature)

```
PLAN_PATH: .claude/plans/{feature}/plan.md
GROUP_ID:  T2-G3                    # opcional — sem ele, full plan
```

- Roda Phase 1 → Phase 9 (test gates).
- Restringe escrita a `scope.create + scope.modify` do group.
- Carrega só os guidelines listados no group.
- Detecta `CONTRACT_MISMATCH` e aborta se o trabalho violaria Section 2.

### Fix Mode (loop de QA)

```
MODE:                 QA_FIX
SECURITY_REPORT_PATH: .claude/reports/security-report-{feature}.md
REVIEW_REPORT_PATH:   .claude/reports/code-review-{feature}.md
SCOPE:                {file list}
```

- **Ordem fixa**: aplica primeiro os auto-fixes de segurança (before/after
  literal do report), depois os BLOCKERs do code review (raciocínio sobre
  o guideline referenciado).
- Não modifica nada fora do `SCOPE`.
- Roda os mesmos test gates do Standard Mode.

---

## 10. Os test gates — barreira final em cada teammate

Antes de retornar SUCCESS, todo `@implementation` (Standard ou Fix) corre,
em ordem, com até 3 tentativas de retry:

```
1. gofmt -l .                              # tem que sair em branco
2. go vet ./...
3. go test ./... -count=1
4. go test ./... -run Integration -v       # se integration tests mudaram
5. go build ./cmd/jitctx                   # binário ainda compila?
```

Se o plano editou guidelines:

```
cd scripts && go run main.go ../.claude/guidelines/*.yml
```

Falhas viram **plan deviations** no Phase 10 summary — o `agent-manager`
e o `qa-coordinator` veem isso, decidem se entra em loop de fix ou se vira
deferred finding.

---

## 11. Por que esse desenho funciona

1. **Hierarquia clara, fan-out controlado.** Manager orquestra,
   QA-coordinator orquestra dentro do QA. Ninguém mais delega. Isso evita
   o anti-padrão "mesh de agentes" onde cada um chama qualquer um.

2. **Paralelismo declarativo, não imperativo.** O paralelismo está no
   plano (Section 9), não no código de orquestração. Se eu mudar a
   topologia da feature, o `agent-manager` reage automaticamente — não
   é preciso reescrever orquestração.

3. **Frozen Domain Contract como ponto de sincronização.** Tier 1 publica
   o contrato. Tiers 2-6 consomem. Qualquer divergência interrompe o
   pipeline e volta à discovery — em vez de virar gambiarra distribuída
   entre N teammates.

4. **Skills + guidelines + plans são todos artefatos de texto.** Tudo que
   um agente "sabe" está em arquivo legível. Reproduzir, debugar e auditar
   o que aconteceu é só `git log` + leitura.

5. **QA é parte do pipeline, não um portão.** Roda sempre, não bloqueia,
   mas registra deferred findings no summary final. O usuário recebe a
   verdade — feature pronta com X issues abertos — não um falso "tudo verde".

6. **Modelos certos no lugar certo.** Opus para arquitetura e auditoria de
   segurança. Sonnet para implementation. Haiku para commit. Custo e
   latência sob controle sem sacrificar qualidade onde ela importa.

7. **Tudo se encaixa em um único codebase Go.** Sem servidor, sem banco,
   sem split front/back — todo o paralelismo vem do DAG de arquivos. O
   pipeline é honesto sobre a forma do projeto.

---

## 12. Como rodar (resumido)

```bash
# Provê requirements.md ou .feature como input
claude --agent agent-manager
```

O resto é autônomo. A única pausa é Step 2 (aprovação do `plan.md`).
Após o commit, o resumo final lista:

- Path do plano.
- Files created/modified, agrupados por tier.
- Paths dos relatórios de QA + verdict.
- SHA do commit.
- Deferred findings, failed groups, skipped tiers (se houver).
