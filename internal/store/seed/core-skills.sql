-- gem2-lfs seed data: 12 L1 core TPMN skills
-- Generated from .gem-squared/reference/gem2-core-skills-L1/
-- Use INSERT OR IGNORE for idempotent seeding.

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-archive-work',
  'gem2-studio',
  'prompt',
  'archive-work SKILL.md',
  '---
name: archive-work
description: >
  (Who) AI agent (e.g., Claude Code).
  (What) Finalize WP — write STATUS/STATE to header, move to archive/, git commit.
  L1 DEFAULT adds task sync, broadcast, proven contract storage, and T-preservation analysis.
  (When) After /verify-work SUCCESS, or anytime human explicitly demands.
  (Where) WP file in .gem-squared/work-plan/ -> .gem-squared/archive/ (L0);
  gem2-lfs via gem2_task_update + gem2_msg_create + gem2_knowledge_create + gem2_delta (L1 DEFAULT).
  (Why) Atomic lifecycle closure — single skill owns all archival side effects.
  L1 adds remote state sync, broadcast, proven contract persistence, and plan-vs-execution delta analysis.
argument-hint: "[WP path or work title]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Edit
  - Bash(git *)
  - Bash(mv *)
  - Bash(mkdir *)
  - Bash(date *)
  - gem2_task_update
  - gem2_msg_create
  - gem2_knowledge_create
  - gem2_delta
---

(* TPMN SKILL — archive-work — L1 *)

(* === Layers === *)
L0 ≜ "Derive WP-level STATUS/STATE, move to archive/, git commit, update alarm.md.
      Fully local — git + .gem-squared/ files"
L1 ≜ "DEFAULT: L0 + gem2_task_update + gem2_msg_create broadcast + gem2_knowledge_create
      (proven contracts) + gem2_delta (T-preservation analysis).
      FALLBACK: degrade to L0 when gem2-studio MCP is unreachable"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "AI agent — auto-proceeds through steps, but ALWAYS asks human permission before file move (step 4)",
  what:  "Finalize WP: derive STATUS/STATE from per-unit fields, write to header,
          move WP to archive/, git commit, update alarm.md.
          L1 DEFAULT: additionally sync task state, broadcast, persist proven contracts, measure T-preservation",
  when:  "After /verify-work with overall_state = SUCCESS (CASE_1, recommended path), or
          anytime human explicitly demands archival regardless of state (CASE_2)",
  where: "L0: .gem-squared/work-plan/{WP}.md -> .gem-squared/archive/{WP}.md + alarm.md + git.
          L1 DEFAULT: additionally gem2-lfs via gem2_task_update, gem2_msg_create, gem2_knowledge_create, gem2_delta",
  why:   "Atomic lifecycle closure — single skill owns ALL archival side effects.
          L1 adds: remote task finalization, cross-session broadcast, proven contract accumulation
          (feeds future /plan-work pattern retrieval), and delta analysis to measure plan-vs-execution fidelity"
]

(* === Input === *)
A ≜ [
  project_slug: 𝕊,
  wp_path: Path?,                      (* direct path to WP file, or ⊥ *)
  work_title: 𝕊?,                     (* search term if wp_path not given *)
  force: 𝔹?,                          (* ⊥ = false. true = case 2, human explicitly demands *)
  task_id: 𝕊?,                        (* L1: remote task ID from gem2-lfs, for gem2_task_update *)
  session_context: 𝕊?                 (* L1: core takeaways from session, for gem2_delta *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  wp_path: Path,                       (* original path *)
  archive_path: Path,                  (* new path in .gem-squared/archive/ *)
  wp_id: 𝕊,
  wp_status: {COMPLETED, ABORTED},    (* WP-level STATUS — written to header *)
  wp_state: {SUCCESS, FAILURE, —},    (* WP-level STATE — written to header *)
  trigger: {CASE_1, CASE_2},          (* which trigger path was used *)
  git_commit: 𝕊,                     (* commit hash *)
  layer: {DEFAULT, FALLBACK},          (* which execution path was taken *)
  survival_score: ℝ[0,100]?,          (* L1 DEFAULT only: gem2_delta T-preservation score *)
  delta_verdict: {SURVIVED, DEGRADED, FAILED}?,  (* L1 DEFAULT only: gem2_delta verdict *)
  archived_at: 𝕊                     (* ISO8601 *)
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ (wp_path ≠ ⊥ ∨ work_title ≠ ⊥)
    ∧ ".gem-squared/alarm.md" exists
    ∧ (force = ⊤                                          (* case 2: no precondition on unit status *)
       ∨ all unit STATUS ∈ {COMPLETED, ABORTED})          (* case 1: all units terminal *)

(* === Trigger Cases === *)
CASE_1 ≜ [
  condition: wp.STATUS = COMPLETED ∧ wp.STATE = SUCCESS,
  source: "/verify-work routing (recommended path)",
  human_permission: required (confirm only)
]

CASE_2 ≜ [
  condition: human explicitly demands archive (any STATUS, any STATE),
  source: "human override — priorities shifted, work abandoned, or FAILURE accepted",
  human_permission: required (confirm + acknowledge non-SUCCESS state)
]

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER execute work — that is /proceed-work mandate,
  ⊢ NEVER verify — that is /verify-work mandate,
  ⊢ NEVER plan — that is /plan-work mandate,
  ⊢ NEVER modify work output or per-unit Result fields — archive what exists as-is,
  ⊢ NEVER archive without human permission — always ask first,
  ⊢ NEVER block on gem2-studio MCP failure — degrade to FALLBACK silently,
  ⊢ NEVER auto-trigger CASE_2 from routing — CASE_2 is human-initiated only
]

(* === Transform === *)
F ≜ <<
  1. Record archived_at timestamp.
     Identify WP:
       IF wp_path provided → read that WP.
       IF only work_title → search work-plan/ for matching WP.
     Determine trigger case:
       IF all units COMPLETED/ABORTED AND all State = SUCCESS → CASE_1.
       ELSE → CASE_2 (requires force=⊤ or explicit human demand).

  2. IF CASE_2 — mark unfinished units:
       FOR each unit with STATUS ∈ {PENDING, IN_PROGRESS}:
         Mark STATUS → ABORTED in WP file.
         Write `- Result: Aborted by human override (archived before completion).`
         Write `- State:` (leave empty — never verified).

  3. Derive WP-level STATUS and STATE from per-unit fields:
       WP STATUS:
         IF all units STATUS = COMPLETED → wp_status = COMPLETED.
         IF any unit STATUS = ABORTED → wp_status = ABORTED.
       WP STATE:
         IF no per-unit State fields filled → wp_state = — (verification was not run).
         IF all filled State = SUCCESS → wp_state = SUCCESS.
         IF any filled State = FAILURE → wp_state = FAILURE.
     Write to WP header:
       `**STATUS:** {wp_status} | **STATE:** {wp_state} | **task_id:** {task_id}`

  4. Ask human permission:
       CASE_1: Show: "Archive COMPLETED|SUCCESS WP: {title}? (move to archive/)"
       CASE_2: Show: "Force-archive WP: {title}? STATUS={wp_status}, STATE={wp_state}.
               {N} units aborted. This is irreversible — confirm?"
       IF denied → STOP without archiving.

  5. Move WP file to archive:
       mkdir -p .gem-squared/archive/
       mv .gem-squared/work-plan/{WP-file}.md → .gem-squared/archive/{WP-file}.md
       Record archive_path.

  6. Git commit:
       git add .gem-squared/archive/{WP-file}.md .gem-squared/work-plan/ .gem-squared/alarm.md
       git commit with message:
         "Archive {wp_status}|{wp_state}: {WP title}

         Trigger: {CASE_1 or CASE_2}
         STATUS: {wp_status} | STATE: {wp_state}
         Archived: .gem-squared/archive/{WP-file}.md

         Date: {date}
         Author: David Seo of GEM².AI"
       Capture git_commit hash.

     (* --- L1 DEFAULT: sync task state after git commit --- *)
       DEFAULT (gem2-studio MCP reachable):
         Call gem2_task_update(
           task_id   = task_id,
           status    = IF wp_status = COMPLETED THEN "COMPLETED" ELSE "CANCELLED",
           result_summary = "{wp_state}: summary of unit results",
           state     = wp_state
         ).
       FALLBACK: skip — git commit is the baseline.

  7. Update alarm.md:
       Decrement IN_PROGRESS (or PENDING) counter.
       IF wp_status = COMPLETED → increment COMPLETED counter.
       IF wp_status = ABORTED → increment ABORTED counter.
       Move from Active Tasks to Recently COMPLETED table:
         | {wp_id} | {date} | {title} | {wp_state} |
       Update timestamp.

     (* --- L1 DEFAULT: broadcast + proven contract storage + delta analysis --- *)
       DEFAULT (gem2-studio MCP reachable):

         (* Broadcast archival event *)
         Call gem2_msg_create(
           from_role     = "ARCHITECT",
           to_role       = "BROADCAST",
           project_slug  = project_slug,
           message       = "Archived {wp_status}|{wp_state}: {WP title} [{CASE_1/CASE_2}]"
         ).

         (* Persist proven contracts for future pattern retrieval *)
         IF wp_state = SUCCESS:
           Call gem2_knowledge_create(
             entity_type   = "contract",
             title         = "Proven: {WP title}",
             content       = CONTRACTs + results summary (all unit CONTRACTs and their Results),
             project_slug  = project_slug,
             tags          = ["proven"]
           ).

         (* T-preservation analysis: plan vs execution fidelity *)
         Collect planned_CONTRACTs = all unit CONTRACT texts as originally written.
         Collect actual_Results = all unit Result texts as recorded by /proceed-work.
         Call gem2_delta(
           original          = planned_CONTRACTs (concatenated),
           transformed       = actual_Results (concatenated),
           session_context   = session_context ∨ "Archiving WP: {WP title}"
         ).
         Receive DeltaReport: survival_score (0-100), verdict (SURVIVED|DEGRADED|FAILED),
           lost/injected/drifted/diluted clause details.
         Set B.survival_score = DeltaReport.survival_score.
         Set B.delta_verdict = DeltaReport.verdict.
         Record survival_score in archive (append to archived WP file footer):
           `<!-- T-preservation: {survival_score}/100 ({delta_verdict}) -->`
         Set layer = DEFAULT.

       FALLBACK (gem2-studio MCP unreachable):
         B.survival_score = ⊥.
         B.delta_verdict = ⊥.
         Set layer = FALLBACK.
         (* git commit + alarm.md is the complete L0 baseline *)

     Output B.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ NEVER execute work — that is /proceed-work mandate,
  ⊢ NEVER verify — that is /verify-work mandate,
  ⊢ NEVER plan — that is /plan-work mandate,
  ⊢ NEVER modify work output or per-unit Result fields — archive what exists as-is,
  ⊢ NEVER archive without human permission — always ask first,
  ⊢ CASE_2 ONLY when human explicitly demands — never auto-triggered by routing,
  ⊢ Git commit happens BEFORE alarm.md and gem2-studio updates — code is committed first,
  ⊢ File move is atomic: WP exists in EITHER work-plan/ OR archive/, never both,
  ⊢ gem2-studio MCP sync failure does NOT block archiving — git commit + alarm.md is the baseline,
  ⊢ gem2_delta failure does NOT block archiving — survival_score = ⊥ is acceptable
]

(* === Invariant === *)
INV ≜ [
  ⊢ This is the ONLY skill that writes WP-level STATUS and STATE to the WP header,
  ⊢ This is the ONLY skill that moves WPs from work-plan/ to archive/,
  ⊢ WP-level STATUS derived from per-unit STATUS — not independently assigned,
  ⊢ WP-level STATE derived from per-unit State — not independently assigned,
  ⊢ wp_state = — is valid (verification was skipped),
  ⊢ CASE_2 marks all unfinished units ABORTED before deriving WP-level fields,
  ⊢ After archiving: work-plan/ contains only active/pending WPs,
  ⊢ After archiving: archive/ contains only terminal WPs (no further modification),
  ⊢ All side effects in one skill: WP header, file move, git, alarm.md, gem2-studio sync,
  ⊢ Proven contracts (SUCCESS) stored as knowledge (DEFAULT) — feeds future /plan-work pattern retrieval,
  ⊢ T-preservation score measures plan-to-execution fidelity (DEFAULT) — tracks quality over time,
  ⊢ Same archived_at timestamp on all writes — enables recency comparison,
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ DEFAULT: git + alarm.md + gem2_task_update + gem2_msg_create broadcast + gem2_knowledge_create + gem2_delta,
  ⊢ FALLBACK: git + alarm.md only — fully functional without gem2-studio MCP,
  ⊢ MANDATE BOUNDARY: lifecycle closure (STATUS/STATE derivation, file move, commit, sync) — NOT work execution, NOT verification, NOT planning
]

(* === Pre-Execution Dialog === *)
Ask_Human ≜ <<
  [field: "confirm",
   prompt: "Archive WP? (shows trigger case, STATUS, STATE, unit summary)",
   required: ⊤]
>>

(* === Post-Execution Routing === *)
Routing ≜ [
  archived = ⊤ ∧ delta_verdict = SURVIVED  → G₁₁: /check-session (clean cycle complete),
  archived = ⊤ ∧ delta_verdict = DEGRADED  → G₁₂: /check-session (cycle complete — note degradation for retrospective),
  archived = ⊤ ∧ delta_verdict = FAILED    → G₁₃: /check-session (cycle complete — flag significant plan-execution drift),
  archived = ⊤ ∧ delta_verdict = ⊥         → G₁₄: /check-session (FALLBACK — no delta available),
  archived = ⊥                              → G₁₅: report failure to human
]
',
  '["core-skill", "archive-work", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-check-session',
  'gem2-studio',
  'prompt',
  'check-session SKILL.md',
  '---
name: check-session
description: >
  (Who) AI agent (e.g., Claude Code).
  (What) Read-only session status — counters, active/pending WPs, blockers, recent messages.
  (When) After /init-session, or anytime human asks for status.
  (Where) alarm.md + work-plan/ + git log (L0); gem2-lfs via gem2_status + gem2_session_context (L1 DEFAULT).
  (Why) Project state visibility across local filesystem AND persistent backend.
argument-hint: "(no arguments — reads current state automatically)"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Bash(git *)
  - Bash(date *)
  - gem2_status
  - gem2_session_context
---

(* TPMN SKILL — check-session — L1 *)

(* === Layers === *)
L0 ≜ "Read-only status from filesystem: alarm.md + work-plan/*.md + git log -5"
L1 ≜ "DEFAULT: merge local state with gem2-lfs (gem2_status + gem2_session_context).
      FALLBACK: degrade to L0 when gem2-lfs is unreachable"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "AI agent — auto-proceed, no human input required",
  what:  "Pure status read: counters, active/pending WPs, blockers, recent messages, recent commits",
  when:  "After /init-session, or on explicit human request (''status'', ''where was I?'')",
  where: "L0: .gem-squared/alarm.md + .gem-squared/work-plan/*.md + git log.
          L1 DEFAULT: additionally gem2-lfs via gem2_status(project_slug) + gem2_session_context(role, project_slug)",
  why:   "Single skill for project state visibility — read-only, zero side effects.
          L1 adds remote task counters, blockers, and cross-session messages that L0 cannot see"
]

(* === Input === *)
A ≜ [
  project_slug: 𝕊,
  session_started_at: 𝕊,              (* ISO8601 — from init-session''s alarm.md timestamp *)
  alarm_path: ".gem-squared/alarm.md",
  work_plan_dir: ".gem-squared/work-plan/",
  role: 𝕊?                            (* L1: role for gem2_session_context, default "ARCHITECT" *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  checked_at: 𝕊,                      (* ISO8601 — when this check was performed *)
  layer: {DEFAULT, FALLBACK},          (* which execution path was taken *)
  counters: 𝕊,                        (* e.g., "PENDING:2 | IN_PROGRESS:1 | COMPLETED:55 | DECOMPOSED:0 | ABORTED:0" *)
  active_works: Seq(𝕊)?,              (* WP titles currently IN_PROGRESS *)
  pending_works: Seq(𝕊)?,             (* WP titles waiting to start *)
  recent_commits: Seq(𝕊)?,            (* last 5 git commits — session context *)
  remote_tasks: Seq(𝕊)?,              (* L1 DEFAULT only: active tasks from gem2-lfs *)
  blockers: Seq(𝕊)?,                  (* L1 DEFAULT only: blockers from gem2-lfs *)
  recent_messages: Seq(𝕊)?,           (* L1 DEFAULT only: recent cross-session messages *)
  recent_decisions: Seq(𝕊)?,          (* L1 DEFAULT only: recent architectural decisions *)
  divergence: 𝕊?                      (* L1 DEFAULT only: local vs remote state mismatch description, ⊥ if consistent *)
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ alarm_path exists

(* === Transform === *)
F ≜ <<
  1. Read local state (ALWAYS — both DEFAULT and FALLBACK):
       Read alarm.md → parse STATUS counter line.
       Scan work-plan/*.md → extract STATUS from each WP file header.
       Classify: active_works (IN_PROGRESS), pending_works (PENDING).
       git log --oneline -5 → recent_commits (session context for human).

  2. Layer switch — attempt DEFAULT, degrade to FALLBACK on failure:
       DEFAULT (gem2-lfs reachable):
         Call gem2_status(project_slug) → receive remote task counters, active tasks, blockers.
         Call gem2_session_context(role ∨ "ARCHITECT", project_slug) → receive recent messages, decisions.
         Merge: compare local counters (step 1) with remote counters.
           IF mismatch → set divergence to description of difference.
           IF consistent → divergence = ⊥.
         Populate: remote_tasks, blockers, recent_messages, recent_decisions from gem2-lfs responses.
         Set layer = DEFAULT.
       FALLBACK (gem2-lfs unreachable or call fails):
         remote_tasks = ⊥, blockers = ⊥, recent_messages = ⊥, recent_decisions = ⊥, divergence = ⊥.
         Set layer = FALLBACK.

  3. Output B as human-readable summary:
       Show: layer, counters, active_works, pending_works, recent_commits.
       IF layer = DEFAULT → additionally show: remote_tasks, blockers, recent_messages, recent_decisions, divergence.
       IF layer = FALLBACK → note: "(gem2-lfs unreachable — showing local state only)".
>>

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER modifies any file — strictly read-only,
  ⊢ NEVER starts or resumes work — status reporting only,
  ⊢ NEVER calls gem2_task_create or gem2_msg_create — read operations only,
  ⊢ NEVER writes to gem2-lfs — only reads from it,
  ⊢ NEVER blocks on gem2-lfs failure — degrades to FALLBACK silently
]

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ Strictly read-only — NEVER modifies any file,
  ⊢ NEVER starts or resumes work — status reporting only,
  ⊢ Local state read (step 1) executes unconditionally — never skipped,
  ⊢ gem2-lfs failure MUST degrade to FALLBACK — never error out
]

(* === Invariant === *)
INV ≜ [
  ⊢ Zero side effects — filesystem unchanged after execution,
  ⊢ Output is current snapshot — no caching between invocations,
  ⊢ Output is self-sufficient — human can make routing decisions from B alone,
  ⊢ git log provides session context,
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ MANDATE BOUNDARY: status reporting — NOT work execution, NOT state mutation, NOT broadcasting
]

(* === Post-Execution Routing === *)
Routing ≜ [
  active_works ≠ ∅     → G₁₁: /proceed-work (resume interrupted work),
  pending_works ≠ ∅    → G₁₂: ask human which to proceed,
  blockers ≠ ∅         → G₁₃: surface blockers to human for resolution,
  divergence ≠ ⊥       → G₁₄: surface divergence to human — "local and remote state differ",
  all_clear = ⊤        → G₁₅: /plan-work or ask human for new work
]
',
  '["core-skill", "check-session", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-end-session',
  'gem2-studio',
  'prompt',
  'end-session SKILL.md',
  '---
name: end-session
description: >
  (Who) AI agent (e.g., Claude Code).
  (What) Commit session state for next-session recovery; broadcast session summary to other sessions.
  (When) Before ending session, or "done for now" / "save session".
  (Where) alarm.md + git commit (L0); additionally gem2-lfs via gem2_msg_create broadcast (L1 DEFAULT).
  (Why) Git commit = primary handoff. L1 adds cross-session broadcast so other sessions see the summary.
argument-hint: "(no arguments — reads current state automatically)"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Edit
  - Bash(git *)
  - Bash(date *)
  - gem2_msg_create
---

(* TPMN SKILL — end-session — L1 *)

(* === Layers === *)
L0 ≜ "Derive summary → update alarm.md → git commit. Git commit is the sole handoff artifact"
L1 ≜ "DEFAULT: after git commit, broadcast session summary via gem2_msg_create to gem2-lfs.
      FALLBACK: degrade to L0 — git commit is the sole handoff, no broadcast"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "AI agent — auto-proceed, no human input required",
  what:  "Record session end: derive summary, update alarm.md timestamp, git commit, optionally broadcast",
  when:  "Before ending session — ''done for now'', ''save session'', ''end session''",
  where: "L0: .gem-squared/alarm.md → git commit.
          L1 DEFAULT: additionally gem2-lfs via gem2_msg_create(to_role=BROADCAST)",
  why:   "Git commit is the primary handoff — self-contained recovery for next session.
          L1 broadcast ensures other active sessions (or future sessions) see the summary
          without having to read git log"
]

(* === Input === *)
A ≜ [
  project_slug: 𝕊,
  alarm_path: ".gem-squared/alarm.md",
  work_plan_dir: ".gem-squared/work-plan/"
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  ended_at: 𝕊,                        (* ISO8601 — session end timestamp *)
  layer: {DEFAULT, FALLBACK},          (* which execution path was taken *)
  git_commit: 𝕊?,                     (* commit hash, ⊥ if working tree was clean *)
  broadcast_sent: 𝔹,                  (* L1 DEFAULT: ⊤ if gem2_msg_create succeeded; FALLBACK: ⊥ *)
  summary: [
    accomplished: Seq(𝕊),             (* what was done this session *)
    active_wps: Seq(𝕊)?,             (* WPs still IN_PROGRESS *)
    pending_decisions: Seq(𝕊)?,      (* decisions needed before next action *)
    next_actions: Seq(𝕊)             (* recommended first steps for next session *)
  ]
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ alarm_path exists

(* === Transform === *)
F ≜ <<
  1. Record ended_at timestamp (ALWAYS — both DEFAULT and FALLBACK):
       Read alarm.md → extract current counters, active WPs.
       Scan work-plan/*.md → identify any IN_PROGRESS or recently modified WPs.
       Check git status → determine if working tree has uncommitted changes.

  2. Derive session summary (ALWAYS):
       accomplished: what tasks were completed or progressed this session
         (from git log today + local WP Result fields written today).
       active_wps: WPs still IN_PROGRESS (from alarm.md).
       pending_decisions: any routing decisions left unresolved
         (e.g., "unit 6 FAILURE — implement, amend, or accept?").
       next_actions: recommended first steps for next session
         (derived from routing state of active WPs).

  3. Update alarm.md (ALWAYS):
       Update "Last checked" timestamp.
       Update footer timestamp line with session summary.
       Do NOT modify counters or task lists — that is /archive-work mandate.

  4. Git commit — PRIMARY HANDOFF (ALWAYS):
       IF working tree is dirty (including alarm.md update from step 3):
         git add .gem-squared/alarm.md (+ any other uncommitted session changes).
         git commit with message:
           "Session end: {1-2 sentence summary}

           Accomplished: {bullet list}
           Active WPs: {list or ''none''}
           Pending decisions: {list or ''none''}
           Next actions: {list}

           Date: {date}
           Author: David Seo of GEM².AI"
         Record git_commit hash.
       IF working tree is clean:
         git_commit = ⊥ (nothing to commit).

  5. Layer switch — attempt DEFAULT, degrade to FALLBACK on failure:
       DEFAULT (gem2-lfs reachable):
         Call gem2_msg_create(
           from_role = "ARCHITECT",
           to_role   = "BROADCAST",
           project_slug = project_slug,
           message   = "Session ended: {1-2 sentence summary from step 2}",
           content   = "Accomplished:\n{accomplished bullets}\n\n
                        Active WPs:\n{active_wps or ''none''}\n\n
                        Pending decisions:\n{pending_decisions or ''none''}\n\n
                        Next actions:\n{next_actions bullets}\n\n
                        Git commit: {git_commit or ''clean — nothing committed''}"
         ).
         IF call succeeds → broadcast_sent = ⊤, layer = DEFAULT.
         IF call fails → broadcast_sent = ⊥, layer = FALLBACK.
       FALLBACK (gem2-lfs unreachable or call fails):
         broadcast_sent = ⊥.
         layer = FALLBACK.
         (* Git commit from step 4 is still the primary handoff — no data loss *)

  6. Output B.
>>

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER modifies alarm.md counters or task lists — only timestamps and footer,
  ⊢ NEVER closes or archives WPs — only reports their current state,
  ⊢ NEVER moves WP files — that is /archive-work mandate,
  ⊢ NEVER executes work — session is ending, not starting,
  ⊢ NEVER skips git commit when working tree is dirty — primary state persistence,
  ⊢ NEVER blocks on gem2-lfs failure — degrades to FALLBACK, git commit is sufficient,
  ⊢ NEVER calls gem2_task_create or gem2_session_context — only gem2_msg_create for broadcast
]

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ NEVER modify alarm.md counters or task lists — only timestamps and footer,
  ⊢ NEVER close or archive WPs — only report their current state,
  ⊢ NEVER move WP files — that is /archive-work mandate,
  ⊢ NEVER execute work — session is ending, not starting,
  ⊢ Git commit is MANDATORY if working tree is dirty — primary state persistence,
  ⊢ Commit message must be self-contained — next session reads git log to recover,
  ⊢ gem2-lfs broadcast (step 5) is OPTIONAL — failure does not fail the skill,
  ⊢ Steps 1-4 execute unconditionally — only step 5 has layer switching
]

(* === Invariant === *)
INV ≜ [
  ⊢ After /end-session: git working tree is CLEAN (all changes committed),
  ⊢ Git commit message contains full handoff context (self-contained recovery),
  ⊢ alarm.md timestamp reflects session end time,
  ⊢ Symmetric with /init-session: init reads what end committed,
  ⊢ Next session recovery path: git log → alarm.md → ready (L0); gem2_session_context → ready (L1),
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ B.broadcast_sent = ⊤ only when gem2_msg_create succeeded,
  ⊢ MANDATE BOUNDARY: session state persistence — NOT work execution, NOT WP archival, NOT counter mutation
]

(* === Post-Execution Routing === *)
Routing ≜ [
  session_ended = ⊤  → G₁₁: STOP (session is over, no further skills)
]
',
  '["core-skill", "end-session", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-extract-skill',
  'gem2-studio',
  'prompt',
  'extract-skill SKILL.md',
  '---
name: extract-skill
description: >
  (Who) AI agent (e.g., Claude Code).
  (What) Format a SUCCESS WP or unit-contract into a TPMN SKILL.md and store in .claude/skills/.
  L1 DEFAULT adds skill registration in gem2-lfs, knowledge graph recording, and optional compile validation.
  (When) After /archive-work SUCCESS, or anytime human says "extract this as a skill" or "make this reusable."
  (Where) Source: .gem-squared/archive/{WP}.md -> Target: .claude/skills/{skill-name}/SKILL.md (L0);
  gem2-lfs via gem2_skill_upsert + gem2_knowledge_create + gem2_compile (L1 DEFAULT).
  (Why) Proven CONTRACTs should be installable. This bridges TPMN archive to the .claude/skills/ ecosystem.
  L1 adds cross-project skill discovery, knowledge graph traceability, and structural validation.
argument-hint: "[WP-ID or unit reference — which proven contract to extract]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Write
  - Glob
  - Grep
  - Bash(mkdir *)
  - Bash(date *)
  - gem2_skill_upsert
  - gem2_knowledge_create
  - gem2_compile
---

(* TPMN SKILL — extract-skill — L1 *)

(* === Layers === *)
L0 ≜ "Read proven WP/UC from archive, format as TPMN SKILL.md, write to .claude/skills/.
      Fully local — git + .gem-squared/ files + .claude/skills/ filesystem"
L1 ≜ "DEFAULT: L0 + gem2_skill_upsert (cross-project discovery) + gem2_knowledge_create
      (extraction event in knowledge graph) + gem2_compile (structural validation).
      FALLBACK: degrade to L0 when gem2-studio MCP is unreachable"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "Human or AI Pilot requesting skill extraction from proven archive",
  what:  "Format a SUCCESS WP or unit-contract into a TPMN SKILL.md file.
          L1 DEFAULT: additionally register in gem2-lfs for cross-project discovery,
          record extraction event in knowledge graph, optionally validate structure via gem2_compile",
  when:  "After /archive-work SUCCESS, or explicit human request to make a proven pattern reusable as a skill",
  where: "L0: Source: .gem-squared/archive/{WP}.md → Target: {project_root}/.claude/skills/{skill-name}/SKILL.md (local filesystem only).
          L1 DEFAULT: additionally gem2-lfs via gem2_skill_upsert, gem2_knowledge_create, gem2_compile",
  why:   "Proven CONTRACTs are knowledge. Placing them in .claude/skills/ makes them discoverable by /search-skill
          and loadable by any AI agent that reads the skills directory.
          L1 adds: cross-project skill registry (gem2_skill_upsert), extraction traceability (gem2_knowledge_create),
          and structural validation (gem2_compile) to ensure the extracted skill compiles to a valid MANDATE"
]

(* === Input === *)
A ≜ [
  source_wp: 𝕊,                        (* WP-ID, e.g., "WP-ST-69" *)
  source_unit: ℕ?,                      (* ⊥ = extract entire WP as skill. ℕ = specific unit only *)
  skill_name: 𝕊?,                      (* ⊥ = derive from WP title in kebab-case *)
  project_slug: 𝕊,
  task_id: 𝕊?,                         (* L1: remote task ID from gem2-lfs, for gem2_skill_upsert context *)
  user_id: 𝕊?                          (* L1: user ID for gem2_skill_upsert *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  source_wp: 𝕊,                        (* echoed *)
  source_unit: ℕ?,                      (* echoed *)
  skill_name: 𝕊,                       (* final kebab-case name *)
  skill_path: Path,                     (* e.g., ".claude/skills/deploy-staging/SKILL.md" *)
  extracted_at: 𝕊,                     (* ISO8601 *)
  contract_count: ℕ,                    (* number of unit-contracts included *)
  source_state: {SUCCESS},              (* must be SUCCESS — precondition *)
  format: {TPMN},                       (* always TPMN — this skill does NOT produce prose *)
  layer: {DEFAULT, FALLBACK},           (* which execution path was taken *)
  registered: 𝔹?,                      (* L1 DEFAULT only: ⊤ if gem2_skill_upsert succeeded *)
  kg_recorded: 𝔹?,                     (* L1 DEFAULT only: ⊤ if gem2_knowledge_create succeeded *)
  compile_valid: 𝔹?                    (* L1 DEFAULT only: ⊤ if gem2_compile passed, ⊥ if skipped or failed *)
]

(* === Precondition === *)
P ≜ source_wp ≠ ⊥
    ∧ project_slug ≠ ⊥
    ∧ ".gem-squared/archive/" exists
    ∧ source WP file exists in archive/
    ∧ source WP STATE = SUCCESS
    ∧ (source_unit = ⊥ ∨ source_unit ≤ WP.unit_count)

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER modifies source archive files — read-only on .gem-squared/archive/,
  ⊢ NEVER executes the extracted skill — extraction and formatting only,
  ⊢ NEVER produces prose or market-standard format — always TPMN,
  ⊢ NEVER overwrites existing skill without human permission,
  ⊢ NEVER extracts from non-SUCCESS WPs — only proven CONTRACTs are extractable,
  ⊢ NEVER blocks on gem2-studio MCP failure — degrade to FALLBACK silently,
  ⊢ NEVER calls gem2_truth_filter — that is L2 only (epistemic verification is /verify-by-gem2 mandate),
  ⊢ NEVER auto-triggers itself from routing — always human-initiated or routed from /archive-work
]

(* === Transform === *)
F ≜ <<
  1. Read and validate source:                                         (ALWAYS — both DEFAULT and FALLBACK)
       Search .gem-squared/archive/ for source WP via Glob/Read.
       Read .gem-squared/archive/{source_wp}.md.
       Parse header: confirm STATUS ∈ {COMPLETED, ABORTED} ∧ STATE = SUCCESS.
       IF source_unit ≠ ⊥ → extract that single unit-contract.
       IF source_unit = ⊥ → extract all unit-contracts.
       For each unit-contract, extract: title, A, B, P, Clarity, Tags, Result, State.
       IF any extracted unit State ≠ SUCCESS → warn human, skip that unit.

  2. Check for existing skill:                                         (ALWAYS — both DEFAULT and FALLBACK)
       Search .claude/skills/ for existing skill via Glob/Read.
         IF match found with relevance = exact → existing skill detected.
           Warn human: "Skill ''{skill_name}'' already exists at {path}. Upsert?"
           IF human denies → STOP, output B with skill_path = existing path.
           IF human confirms → proceed to overwrite (upsert).
         IF no match → proceed to create new skill.

  3. Derive skill name:                                                (ALWAYS — both DEFAULT and FALLBACK)
       IF skill_name provided → use as-is (must be kebab-case).
       IF skill_name = ⊥ → derive from WP title:
         Strip "WP-ST-{N}: " prefix.
         Convert to kebab-case: lowercase, spaces→hyphens, strip special chars.
         e.g., "Deploy v2.3 to Staging" → "deploy-v23-to-staging".

  4. Format as TPMN SKILL.md:                                          (ALWAYS — both DEFAULT and FALLBACK)
       Follow .claude/TPMN-SKILL-STANDARD.md for structure.
       Generate YAML frontmatter:
         name: {skill_name}
         description: >
           (What) {WP objective — one line}
           (When) {derived from WP context — when this pattern applies}
           (Why) {extracted from WP — why this workflow exists}
           (How) {summary of unit-contract sequence}
         metadata:
           author: David Seo of GEM².AI
           version: 1.0.0
           extracted_from: {source_wp}
           extracted_at: {ISO8601}

       Generate TPMN body:
         (* TPMN SKILL — {skill_name} *)
         (* Extracted from {source_wp} — proven SUCCESS *)

         (* === Grounding 5W === *)
         Grounding_5W ≜ [ ... derived from WP context ... ]

         (* === Input === *)
         A ≜ [ ... generalized from unit-contract A fields ... ]

         (* === Output === *)
         B ≜ [ ... generalized from unit-contract B fields ... ]

         (* === Precondition === *)
         P ≜ ... generalized from unit-contract P fields ...

         (* === Transform === *)
         F ≜ <<
           ... unit-contract sequence from WP ...
           Each step corresponds to one unit-contract.
           Result fields included as reference (what was produced when this pattern was proven).
         >>

         (* === Constraint === *)
         CONSTRAINT ≜ [ ... derived from WP constraints and lessons ... ]

         (* === Provenance === *)
         Provenance ≜ [
           source: "{source_wp}",
           state: "SUCCESS",
           proven_at: "{WP archived_at}",
           unit_contracts: {contract_count}
         ]

  5. Write to .claude/skills/ (or upsert if existing):                 (ALWAYS — both DEFAULT and FALLBACK)
       mkdir -p {project_root}/.claude/skills/{skill_name}/
       Write SKILL.md to {project_root}/.claude/skills/{skill_name}/SKILL.md.
       Record skill_path.
       Record extracted_at timestamp.

     (* --- L1 DEFAULT: register skill + record extraction + validate structure --- *)
       DEFAULT (gem2-studio MCP reachable):

         (* Register extracted skill in gem2-lfs for cross-project discovery *)
         Call gem2_skill_upsert(
           user_id       = user_id,
           title         = skill_name,
           content       = SKILL.md content (full text of the generated skill),
           skill_type    = "contract",
           tags          = ["extracted-skill", skill_name, source_wp, "tpmn"]
         ).
         IF success → B.registered = ⊤.
         IF failure → B.registered = ⊥. Log warning but DO NOT abort — L0 baseline is complete.

         (* Record the extraction event in knowledge graph *)
         Call gem2_knowledge_create(
           entity_type   = "contract",
           title         = "Extracted: {skill_name}",
           content       = "Extracted from {source_wp} ({contract_count} unit-contracts). "
                           ∧ "Skill written to {skill_path}. "
                           ∧ "Provenance: proven SUCCESS at {WP archived_at}.",
           project_slug  = project_slug,
           tags          = ["extracted-skill", skill_name, source_wp]
         ).
         IF success → B.kg_recorded = ⊤.
         IF failure → B.kg_recorded = ⊥. Log warning but DO NOT abort — L0 baseline is complete.

         (* Optionally validate the extracted skill compiles to a valid MANDATE *)
         Call gem2_compile(
           content           = extracted SKILL.md (full text),
           session_context   = "Verifying extracted skill structure for: {skill_name}"
         ).
         IF success ∧ compile result = valid → B.compile_valid = ⊤.
         IF success ∧ compile result = invalid → B.compile_valid = ⊥. Warn human: "Extracted skill did not compile cleanly — review structure."
         IF failure → B.compile_valid = ⊥. Log warning but DO NOT abort.
         Set layer = DEFAULT.

       FALLBACK (gem2-studio MCP unreachable):
         B.registered = ⊥.
         B.kg_recorded = ⊥.
         B.compile_valid = ⊥.
         Set layer = FALLBACK.
         (* Skill file is already written — L0 baseline is fully complete *)

     Output B.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ ONLY extracts from SUCCESS WPs — never from FAILURE, PENDING, or IN_PROGRESS,
  ⊢ NEVER modifies the source archive file — read-only on archive/,
  ⊢ ALWAYS produces TPMN format — never prose, never market-standard format,
  ⊢ NEVER executes the extracted skill — extraction and formatting only,
  ⊢ NEVER overwrites an existing skill without human permission,
  ⊢ Generalize where possible — remove project-specific paths/names from A, B, P,
  ⊢ Preserve provenance — source WP, proven date, and STATE always recorded,
  ⊢ gem2-studio MCP sync failure does NOT block extraction — skill file write is the baseline,
  ⊢ gem2_compile failure does NOT block extraction — compile_valid = ⊥ is acceptable,
  ⊢ Each MCP call failure is independent — one failure does NOT prevent other MCP calls
]

(* === Invariant === *)
INV ≜ [
  ⊢ B is state (extraction result), not action — skill produces B and STOPS,
  ⊢ Every extracted skill has a Provenance section — traceable to source WP,
  ⊢ Every extracted skill follows TPMN-SKILL-STANDARD.md structure — not ad-hoc,
  ⊢ Target directory is always .claude/skills/ — the standard skills location,
  ⊢ TPMN format in .claude/skills/ is intentional — TPMN skills are platform-agnostic,
  ⊢ This skill is the WRITE counterpart to /search-skill''s READ of .claude/skills/,
  ⊢ This skill closes the knowledge loop: archive → extract → .claude/skills/ → /search-skill → /plan-work,
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ DEFAULT: skill file write + gem2_skill_upsert + gem2_knowledge_create + gem2_compile,
  ⊢ FALLBACK: skill file write only — fully functional without gem2-studio MCP,
  ⊢ L0 baseline (skill file written to .claude/skills/) is ALWAYS completed before any MCP calls,
  ⊢ MANDATE BOUNDARY: extraction and formatting only — NOT work execution, NOT verification, NOT archiving
]

(* === Post-Execution Routing === *)
(* G_ij bridges — formal handoff contracts *)
Routing ≜ [
  G(extract-skill, ⊥):
    skill_path exists ∧ layer = DEFAULT  → human reviews extracted skill (terminal — no automatic routing),
  G(extract-skill, ⊥):
    skill_path exists ∧ layer = FALLBACK → human reviews extracted skill (terminal — MCP features unavailable),
  G(extract-skill, ⊥):
    extraction_failed → report to human with reason (terminal)
]
',
  '["core-skill", "extract-skill", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-init-session',
  'gem2-studio',
  'prompt',
  'init-session SKILL.md',
  '---
name: init-session
description: >
  (Who) AI agent (e.g., Claude Code).
  (What) Bootstrap session — ensure 8 mandatory files + 3 canonical directory trees exist, record session start, register with gem2-lfs.
  (When) Every session start. Must be first skill triggered.
  (Where) .gem-squared/ + .claude/skills/ + ~/.claude/skills/ (L0); gem2-lfs via gem2_session_context + gem2_task_create (L1 DEFAULT).
  (Why) Infrastructure gate — no work can proceed without mandatory files. L1 adds remote session recovery and task registration.
argument-hint: "(no arguments — bootstraps from project_dir automatically)"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Write
  - Bash(mkdir *)
  - Bash(date *)
  - Bash(git *)
  - gem2_session_context
  - gem2_task_create
---

(* TPMN SKILL — init-session — L1 *)

(* === Layers === *)
L0 ≜ "Bootstrap from filesystem only: ensure 8 mandatory files + 3 canonical directory trees, record timestamp, invoke /skill-to-kg archive"
L1 ≜ "DEFAULT: after L0 bootstrap completes, register session with gem2-lfs (gem2_task_create) and recover prior session state (gem2_session_context).
      FALLBACK: degrade to pure L0 when gem2-lfs is unreachable"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "Any agent starting a new session — must be first skill triggered",
  what:  "Bootstrap session infrastructure — ensure 8 mandatory files + 3 canonical directory trees exist, record session start timestamp, register with gem2-lfs, recover prior session state",
  when:  "Every session start, or after context compaction / memory drift",
  where: "L0: .gem-squared/ + .claude/skills/ + ~/.claude/skills/ — local filesystem only.
          L1 DEFAULT: additionally gem2-lfs via gem2_session_context(role, project_slug) + gem2_task_create(role, title, project_slug)",
  why:   "Infrastructure gate — no work can proceed without mandatory files. Ensures consistent project structure across sessions.
          L1 adds remote session recovery (prior context from gem2-lfs) and task registration (session start tracked as a task)"
]

(* === Input === *)
A ≜ [
  project_dir: Path,
  8_Mandatory_Files: [
    ".claude/skills/{slug}/SKILL.md"       — project skill,
    ".claude/settings.json"                — Claude Code permissions,
    ".claude/TPMN-SKILL-STANDARD.md"       — skill authoring standard,
    ".gem-squared/alarm.md"                — mutable state,
    "CLAUDE.md"                            — behavioral rules,
    ".mcp.json"                            — MCP server config,
    ".gitignore"                           — git hygiene,
    ".gem-squared/work-plan/"              — directory
  ],
  role: 𝕊?                                (* L1: role for gem2-lfs calls, default "ARCHITECT" *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,                   (* sole project identity — derived from SKILL.md name or dir basename *)
  session_started_at: 𝕊,              (* ISO8601 timestamp *)
  mandatory_files: 𝕊,                 (* e.g., "All 8 present" or "Created: alarm.md, .gitignore" *)
  layer: {DEFAULT, FALLBACK},          (* which execution path was taken *)
  remote_session: 𝕊?,                 (* L1 DEFAULT only: recent session context summary from gem2-lfs *)
  task_id: 𝕊?                         (* L1 DEFAULT only: task ID returned by gem2_task_create *)
]

(* === Precondition === *)
P ≜ project_dir ≠ ⊥

(* === Transform === *)
F ≜ <<
  1. Retrieve project_slug (ALWAYS — both DEFAULT and FALLBACK):
       Read .claude/skills/*/SKILL.md → extract `name:` field from frontmatter.
       IF no SKILL.md exists → derive from directory name (basename of project_dir).
       IF project_slug not found by any method → derive from basename — always succeeds.

  2. Ensure all 3 canonical directory trees exist (ALWAYS — both DEFAULT and FALLBACK):
       Tree 1 — .gem-squared/ (20 dirs):
         work-plan, verify-work-logs, truth-logs, archive, evidences, external-skills,
         gem2-core-skills/_preface,
         gem2-core-skills/{archive-work,check-session,end-session,extract-skill,
                          init-session,plan-work,proceed-work,search-kg,
                          search-skill,skill-to-kg,update-work-plan,verify-work}
       Tree 2 — .claude/skills/ (5 dirs):
         agents, {slug}/references, {slug}/assets, {slug}/eval-viewer, {slug}/scripts
       Tree 3 — ~/.claude/skills/ (12 dirs):
         archive-work, check-session, end-session, extract-skill,
         init-session, plan-work, proceed-work, search-kg,
         search-skill, skill-to-kg, update-work-plan, verify-work
       Allowed tool: Bash(mkdir -p {path}) for each missing directory.

  3. FOR each file in 8_Mandatory_Files (ALWAYS — both DEFAULT and FALLBACK):
       check exists.
       IF missing → create with defaults:
         SKILL.md: minimal project identity skeleton (bundled template).
         settings.json: default TPMN permissions (Skill, Read, Write, Edit, Glob, Grep,
           Bash(mkdir *), Bash(date *), Bash(git *), Bash(mv *), Bash(ls *), Bash(uuidgen *)).
         TPMN-SKILL-STANDARD.md: bundled v3.0 fallback.
         alarm.md: empty counters (PENDING:0|IN_PROGRESS:0|COMPLETED:0|DECOMPOSED:0|ABORTED:0).
         CLAUDE.md: bundled template.
         .mcp.json: minimal MCP config.
         .gitignore: standard gem2 ignores.
         work-plan/: mkdir (already covered by step 2).
       IF exists AND NOT updatable (alarm.md, .mcp.json, .gitignore, settings.json):
         NEVER overwrite — these contain mutable state or user customizations.

  4. Record session_started_at timestamp in alarm.md (ALWAYS — both DEFAULT and FALLBACK):
       Update "Last checked" line with current ISO8601 timestamp.

  5. Invoke /skill-to-kg archive — skill hygiene (ALWAYS — both DEFAULT and FALLBACK):
       This moves all non-core skills from .claude/skills/ to .gem-squared/external-skills/.
       BEFORE archiving, MUST print a clear notification to the user:
         "TPMN Skill Hygiene: /skill-to-kg will move {N} non-core skills from
          .claude/skills/ to .gem-squared/external-skills/.
          Skills moved: {list of skill names}.
          These skills are NOT deleted — they remain searchable via /search-kg
          and can be restored anytime with: /skill-to-kg restore <skill-name>"
       IF no non-core skills exist → skip silently.
       IF non-core skills exist → print notification, then archive.
       This ensures .claude/skills/ contains only the 12 lifecycle skills
       + the project identity skill — reducing trigger collision.

  6. Layer switch — attempt DEFAULT, degrade to FALLBACK on failure:
       DEFAULT (gem2-lfs reachable):
         6a. Call gem2_session_context(role ∨ "ARCHITECT", project_slug)
             → receive prior session state (recent context summary, last active work, messages).
             IF gem2-lfs returns session state → populate B.remote_session with summary.
             IF gem2-lfs returns empty / no prior session → B.remote_session = ⊥.
         6b. Call gem2_task_create(role ∨ "ARCHITECT", title="Session init: {project_slug}", project_slug)
             → register session start as a tracked task in gem2-lfs.
             Populate B.task_id with returned task identifier.
         Set B.layer = DEFAULT.
       FALLBACK (gem2-lfs unreachable or any call fails):
         B.remote_session = ⊥, B.task_id = ⊥.
         Set B.layer = FALLBACK.
>>

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER starts or executes work — infrastructure readiness only,
  ⊢ NEVER analyzes session state or produces status reports — that is /check-session mandate,
  ⊢ NEVER overwrites existing mutable files (alarm.md, .mcp.json, .gitignore, settings.json),
  ⊢ NEVER modifies CLAUDE.md beyond initial creation — user behavioral rules are sacred,
  ⊢ NEVER skips mandatory file checks — all 8 verified every time,
  ⊢ NEVER calls gem2_truth_filter — that is L2 only,
  ⊢ NEVER writes to gem2-lfs beyond task registration (gem2_task_create) — no state mutation,
  ⊢ NEVER blocks on gem2-lfs failure — degrades to FALLBACK silently
]

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ project_slug must be resolved before step 2,
  ⊢ NEVER start work — infrastructure readiness only,
  ⊢ NEVER analyze session state — that is /check-session mandate,
  ⊢ NEVER modify alarm.md beyond initial creation + timestamp,
  ⊢ NEVER modify CLAUDE.md beyond initial creation,
  ⊢ NEVER skip file checks — all 8 checked every time,
  ⊢ ALWAYS notify user before /skill-to-kg archive — list skill names and explain they are not deleted,
  ⊢ L0 bootstrap (steps 1–5) executes unconditionally — never skipped regardless of layer,
  ⊢ gem2-lfs failure MUST degrade to FALLBACK — never error out
]

(* === Invariant === *)
INV ≜ [
  ⊢ All 3 canonical directory trees exist after this skill completes (28 dirs total),
  ⊢ All 8 mandatory files exist after this skill completes,
  ⊢ project_slug is the sole project identity — derived from SKILL.md name or dir basename,
  ⊢ Non-updatable files (alarm.md, .mcp.json, .gitignore, settings.json) are NEVER overwritten if they exist,
  ⊢ alarm.md records session_started_at timestamp,
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ L0 steps (1–5) complete successfully regardless of gem2-lfs availability
]

(* === Post-Execution Routing === *)
(* G_ij bridges — formal handoff contracts *)
Routing ≜ [
  G(init-session, ⊥):
    project_slug = ⊥ → STOP (terminal — cannot derive project identity),
  G(init-session, skill-to-kg):
    project_slug ≠ ⊥ → B_init.project_slug feeds A_skill-to-kg.project_slug (skill hygiene archive),
  G(init-session, check-session):
    project_slug ≠ ⊥ → B_init.project_slug + B_init.session_started_at + B_init.layer feeds A_check.project_slug + A_check.session_started_at
]
',
  '["core-skill", "init-session", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-plan-work',
  'gem2-studio',
  'prompt',
  'plan-work SKILL.md',
  '---
name: plan-work
description: >
  (What) Decompose work into <=9 unit-works with CONTRACTs and clarity %.
  (When) New work request, or sub-decomposition of low-clarity units.
  (Why) CONTRACTs before execution. (How) gem2_compile + cross-project search + decompose + write WP.
argument-hint: "[work description or WP path for decomposition]"
metadata:
  author: David Seo of GEM2.AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Write
  - Glob
  - Grep
  - Bash(date *)
  - Bash(uuidgen *)
  - gem2_compile
  - gem2_knowledge_search
  - gem2_semantic_search
  - gem2_task_create
---

(* TPMN SKILL -- plan-work *)

(* === Grounding 5W === *)
Grounding_5W := [
  who:   "Any agent or human needing work decomposition into executable units",
  what:  "Decompose work into <=9 unit-works with CONTRACTs and clarity %",
  when:  "New work request, or sub-decomposition of low-clarity units",
  where: ".gem-squared/work-plan/ (filesystem) + gem2-lfs (remote task registration)",
  why:   "CONTRACTs before execution -- no uncontracted work enters the system"
]

(* === Input === *)
A := [
  work: S,                            (* what needs to be planned -- may be terse or detailed *)
  project_slug: S,
  time_stamp: S,                      (* ISO8601 -- when this planning was triggered *)
  parent_wp: Path?,                   (* bot for top-level, path for sub-decomposition *)
  parent_unit_index: N?               (* which unit in parent is being decomposed *)
]

(* === Output === *)
B := [
  wp_path: Path,                       (* e.g., ".gem-squared/work-plan/WP-ST-57.md" *)
  wp_id: S,                           (* local identifier, e.g., "WP-ST-57" *)
  task_id: S,                         (* L0: local uuid8, L1: gem2-lfs-assigned *)
  created_at: S,                      (* ISO8601 -- recorded in WP file header *)
  unit_count: N,                       (* 1-9 *)
  avg_clarity: N,                      (* 0-100, average across all units *)
  references_used: Seq(S)?,           (* patterns that informed the plan *)
  compile_report: Record?              (* L1 DEFAULT: CompileReport from gem2_compile. bot on FALLBACK *)
]

(* === Negative Contract === *)
-B := [
  |- NEVER execute work -- planning and CONTRACT creation only,
  |- NEVER produce code or artifacts -- WP file is the sole output,
  |- NEVER skip 5W1H gathering for top-level WPs -- sub-decompositions may inherit from parent,
  |- NEVER exceed 9 units per level -- Miller''s law ceiling,
  |- NEVER decide recursion depth -- human decides when clarity % is sufficient,
  |- NEVER call gem2_truth_filter -- that is L2 exclusive mandate
]

(* === Precondition === *)
P := work != bot
    /\ project_slug != bot
    /\ ".gem-squared/work-plan/" exists

(* === Layer === *)
Layer := L1  (* DEFAULT: gem2-lfs + gem2-studio checker tools. FALLBACK: L0 filesystem only. *)

(* === 5W1H Dimensions === *)
(* These 6 dimensions must be resolved before decomposition begins.      *)
(* The AI extracts what is already present in `work`, then asks human    *)
(* only about missing or ambiguous dimensions.                           *)
DIMENSIONS := [
  WHAT:  S,   (* Scope -- what exactly needs to change or be built *)
  WHY:   S,   (* Motivation -- business reason, user need, or technical debt *)
  WHO:   S?,  (* Stakeholders/actors -- who requested, who is affected, who reviews *)
  WHERE: S,   (* Target -- runtime/deployment environment + which files/modules. NEVER tech stack -- that is HOW *)
  WHEN:  S?,  (* Timeline/ordering -- urgency, dependencies, sequencing constraints *)
  HOW:   S?   (* Tech stack + approach -- frameworks, libraries, languages, architecture pattern *)
]
(* WHAT and WHERE are always required -- without them, decomposition is guessing. *)
(* WHY is required for top-level WPs (parent_wp = bot), optional for child WPs.  *)
(* WHO, WHEN, HOW are optional -- ask only if genuinely unclear.                 *)

(* === Transform === *)
F := <<
  1. Gather 5W1H context (smart extraction):
       DEFAULT:
         Call gem2_compile(content=work, session_context=project_slug + time_stamp summary).
         Read CompileReport.mandate: use mandate.A, mandate.B, mandate.P, mandate.neg_B
         to inform 5W1H extraction -- the compiled MANDATE provides formal structure
         that sharpens WHAT (from mandate.B), WHERE (from mandate.A sources),
         WHY (from mandate rationale), and P (constraints that map to WHEN/HOW).
         Parse `work` argument to extract remaining 5W1H dimensions not covered by mandate.
       FALLBACK:
         Parse `work` argument to extract existing 5W1H dimensions (same as L0).
       FOR each dimension in {WHAT, WHY, WHO, WHERE, WHEN, HOW}:
         IF clearly stated in work (or mandate) -> mark as RESOLVED, record extracted value.
         IF partially stated or ambiguous -> mark as UNCLEAR.
         IF absent -> mark as MISSING.
       Determine which dimensions need human input:
         Required: WHAT must be explicit (not just a vague noun).
         Required: WHERE must be identifiable (target environment or file/module scope).
         Required (top-level only): WHY must be stated (not "because user asked").
         Optional: WHO, WHEN, HOW -- ask only if genuinely needed for decomposition.
       IF any required dimension is MISSING or UNCLEAR:
         Ask human for the missing dimensions. Show what was extracted,
         highlight what is missing. Use concise targeted questions, not a form.
         Example: "WHERE: which files/modules does this affect?" (existing codebase)
         Example: "WHERE: what is the target environment -- web, mobile, local, Docker, cloud?" (greenfield)
         Example: "WHY: what problem does this solve or what need does it serve?"
         Example: "HOW: which tech stack/framework/language?" (greenfield)
         NEVER ask tech stack under WHERE -- tech stack is always HOW.
       IF parent_wp != bot (sub-decomposition):
         Inherit WHY, WHO, WHEN from parent WP -- only ask about WHAT, WHERE, HOW
         if the parent unit''s CONTRACT doesn''t already specify them.
       Record resolved 5W1H as context for step 2 (search) and step 3 (decomposition).
  2. Search for relevant patterns:
       DEFAULT:
         Call gem2_knowledge_search(project_slug=project_slug, query=work, entity_type="contract")
         to find proven contracts in gem2-lfs.
         Call gem2_semantic_search(query=work, project_slug=project_slug)
         for cross-project semantic pattern discovery.
         Merge remote results with local filesystem results below.
       FALLBACK:
         (no remote search -- local only)
       ALWAYS (both DEFAULT and FALLBACK):
         Search archive/ + work-plan/ for relevant patterns via tag-based Glob/Grep/Read.
         Read project skill references/ for project-specific context.
       Record references_used from all search results (remote + local).
  3. Decompose into 1-9 unit-works (Miller''s law: 7+-2).
     Use resolved 5W1H context to inform decomposition -- WHERE determines target environment and file scope per unit,
     HOW constrains tech stack and approach, WHEN determines ordering and dependencies.
     DEFAULT: If gem2_compile produced mandate.geometric_decomposition, use it as a
     structural hint for unit boundaries (not a binding constraint -- human-visible decomposition is final).
     For each unit-work define:
       CONTRACT: A -> B | P,
       Clarity %: 0-100,
       Unclear: what is ambiguous if < 100%,
       Tags: 1-3 tags in {verb-ing}-{object} format derived from CONTRACT A/B.
         (* Tag format: /^[a-z]+-[a-z]+(-[a-z]+)*$/ *)
         (* Tags capture intent -- what this unit does, searchable by /search-kg *)
  4. Derive WP number and task_id. Write WP file:
     IF parent_wp = bot -> next WP-ST-{N} from existing files in work-plan/.
     IF parent_wp != bot -> WP-ST-{parent_N}-{child_M}.
     Generate task_id:
       DEFAULT:
         Call gem2_task_create(
           role="ARCHITECT",
           title=WP title,
           project_slug=project_slug,
           description=Objective text,
           tags=aggregated unit tags,
           metadata_json={"wp_id": "WP-ST-{id}", "avg_clarity": N, "unit_count": N}
         ).
         Record task_id from gem2-lfs response.
         IF gem2_task_create fails -> FALLBACK to local uuid8.
       FALLBACK:
         Generate task_id: `uuidgen | cut -c1-8` -> local uuid8.
     Write .gem-squared/work-plan/WP-ST-{id}.md:

       # WP-ST-{id}: {title}
       **STATUS:** PENDING | **STATE:** --- | **task_id:** {task_id}
       **created_at:** {time_stamp} | **project_slug:** {project_slug}

       ## Objective
       {1-3 sentences synthesized from resolved 5W1H:
        WHAT is being done, WHY it matters, WHERE it applies.
        Include HOW/WHEN/WHO only if they constrain the work.}

       ## Unit-Works
       ### 1. {title} | STATUS: PENDING
       - A: {input state}
       - B: {output state}
       - P: {preconditions}
       - Clarity: {N}%
       - Unclear: {what is ambiguous}
       - Tags: [{verb-ing}-{object}, ...] (1-3 tags, searchable by /search-kg)
       - Result: (filled by /proceed-work)
       - State: (filled by /verify-work -- SUCCESS or FAILURE)
       - Truth: (optional external verification -- score% | Alignment | SPT | EEF)

       ### 2. {title} | STATUS: PENDING
       - A: / B: / P: / Clarity: / Unclear:
       - Result:
       - State:
       - Truth:

       (repeat for each unit-work, max 9)

       ## References
       - {patterns used}
     IF parent_wp != bot -> update parent DECOMPOSITION section.
     Calculate avg_clarity. Output B.
>>

(* === Constraint === *)
CONSTRAINT := [
  |- NEVER execute work -- planning and CONTRACT creation only,
  |- NEVER exceed 9 sub-works per level -- Miller''s law ceiling,
  |- NEVER decompose without resolving WHAT and WHERE -- guessing scope produces bad CONTRACTs. WHERE = target + location, never tech stack,
  |- NEVER ask about dimensions already clearly stated in work arg -- extract, don''t interrogate,
  |- NEVER skip 5W1H for top-level WPs -- sub-decompositions may inherit from parent,
  |- NEVER decide recursion depth -- human decides when clarity % is sufficient,
  |- PREFER gem2-lfs patterns when available (DEFAULT) -- fall back to filesystem search (FALLBACK),
  |- gem2-lfs/gem2-studio unavailability does NOT block planning -- FALLBACK is fully functional
]

(* === Invariant === *)
INV := [
  |- Every WP Objective section reflects resolved 5W1H context -- not just a rephrased work arg,
  |- Every unit-work has a CONTRACT (A -> B | P) -- no uncontracted work,
  |- Every unit-work has a clarity % -- no unassessed scope,
  |- Every unit-work has 1-3 Tags in {verb-ing}-{object} format -- searchable by /search-kg,
  |- Every unit-work has its own STATUS line (PENDING -> IN_PROGRESS -> COMPLETED/ABORTED),
  |- Every unit-work has a Result field (empty until /proceed-work fills it),
  |- WP-level STATUS derived from unit statuses: all COMPLETED -> COMPLETED, any IN_PROGRESS -> IN_PROGRESS,
  |- WP file is ALWAYS written (L0 baseline) -- it is the source of truth,
  |- DEFAULT: dual persistence -- WP file + gem2-lfs task, bidirectional link (task_id <-> wp_id),
  |- FALLBACK: single persistence -- WP file only, task_id is local uuid8,
  |- DEFAULT: gem2_compile report informs but does not override human-facing decomposition,
  |- Child WPs reference parent, parent tracks children -- tree is navigable,
  |- MANDATE BOUNDARY: plan-work decomposes and writes WP -- it does NOT execute, verify, or archive
]

(* === Pre-Execution Dialog === *)
Ask_Human := <<
  (* Step 0: If work arg is empty or bot, ask for initial description. *)
  [field: "work",
   prompt: "What work needs to be planned?",
   required: T,
   condition: work = bot]

  (* Step 1 (5W1H): After parsing work arg, ask only about MISSING/UNCLEAR dimensions. *)
  (* Show extracted values, then targeted questions for gaps.                            *)
  [field: "5w1h_gaps",
   prompt: "I extracted the following from your request:
            {show RESOLVED dimensions}
            Please clarify:
            {for each MISSING/UNCLEAR required dimension: targeted question}",
   required: T,
   condition: E required dimension in {WHAT, WHERE, WHY} that is MISSING \/ UNCLEAR]
>>

(* === Post-Execution Routing === *)
(* G_ij bridges -- formal handoff contracts *)
Routing := [
  G(plan-work, proceed-work):
    avg_clarity >= 70 -> B_plan.wp_path feeds A_proceed.wp_path,
  G(plan-work, plan-work):
    avg_clarity < 70 -> suggest further decomposition or ask human for clarification,
  G(plan-work, parent-context):
    parent_wp != bot -> return to parent context
]
',
  '["core-skill", "plan-work", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-proceed-work',
  'gem2-studio',
  'prompt',
  'proceed-work SKILL.md',
  '---
name: proceed-work
description: >
  (What) Execute ONE unit-work -- fulfill CONTRACT, record Result, verify inline, retry on FAILURE.
  (When) After /plan-work, or resuming from /check-session. One unit per invocation.
  (Why) Produce verified results. (How) Find PENDING unit -> bridge validate -> execute -> verify -> next.
argument-hint: "[WP path or work title]"
metadata:
  author: David Seo of GEM2.AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
  - gem2_task_update
  - gem2_bridge
---

(* TPMN SKILL -- proceed-work *)

(* === Grounding 5W === *)
Grounding_5W := [
  who:   "Agent executing planned unit-works on behalf of human",
  what:  "Execute ONE unit-work -- fulfill CONTRACT.B given CONTRACT.A under CONTRACT.P, record Result, verify inline via /verify-work",
  when:  "After /plan-work produces a WP, or resuming from /check-session with active/pending WPs",
  where: ".gem-squared/work-plan/{WP}.md (read CONTRACT, write Result+Status) + gem2-lfs (remote task sync)",
  why:   "Execution produces verified results -- the only path from plan to proven CONTRACT"
]

(* === Input === *)
A := [
  project_slug: S,
  wp_path: Path?,                      (* direct path to WP file, or bot *)
  work_title: S?,                     (* search term if wp_path not given *)
  unit_index: N?                      (* specific unit to execute, or bot = first PENDING *)
]

(* === Output === *)
B := [
  project_slug: S,
  wp_path: Path,
  wp_id: S,
  unit_index: N,                      (* which unit was executed, 1-based *)
  unit_title: S,
  unit_status: {COMPLETED, ABORTED},  (* this unit''s STATUS *)
  unit_state: {SUCCESS, FAILURE}?,    (* this unit''s STATE from /verify-work. bot if ABORTED *)
  retried: B_bool,                    (* true if unit was retried after initial FAILURE *)
  bridge_verdict: {VALID, PARTIAL, INVALID, SKIPPED}?,  (* L1 DEFAULT: inter-unit handoff check. bot on FALLBACK or unit_index=1 *)
  units_done: N,                      (* total completed units in this WP *)
  units_total: N,
  wp_status: {IN_PROGRESS, COMPLETED, ABORTED},  (* derived from all unit statuses *)
  updated_at: S,                      (* ISO8601 *)
  output_summary: S                   (* what was accomplished in this unit *)
]

(* === Negative Contract === *)
-B := [
  |- NEVER produces a plan or WP file -- that is /plan-work mandate,
  |- NEVER writes State field directly -- /verify-work owns STATE determination,
  |- NEVER archives -- that is /archive-work mandate,
  |- NEVER modifies CONTRACTs -- execution works against the planned CONTRACT, not a revised one,
  |- NEVER batches multiple units -- ONE unit per invocation,
  |- NEVER calls gem2_truth_filter -- that is L2 exclusive mandate
]

(* === Precondition === *)
P := project_slug != bot
    /\ (wp_path != bot \/ work_title != bot)
    /\ ".gem-squared/work-plan/" exists

(* === Layer === *)
Layer := L1  (* DEFAULT: gem2-lfs task sync + gem2-studio bridge validation. FALLBACK: L0 filesystem only. *)

(* === Transform === *)
F := <<
  1. Record started_at timestamp. Identify WP:
       IF wp_path provided -> read that WP.
       IF only work_title -> search work-plan/ for matching WP.
       IF neither -> present active/pending works from alarm.md.
       DEFAULT:
         Call gem2_task_update(task_id=WP.task_id, status="IN_PROGRESS").
         IF gem2_task_update fails -> continue (WP file is source of truth).
       FALLBACK:
         (no remote task sync)
  2. Find target unit-work:
       IF unit_index provided -> use that unit.
       ELSE -> find first unit with STATUS: PENDING in WP file.
       IF no PENDING units remain -> wp_status=COMPLETED, output B, STOP.
  3. Ask human permission:
       Show: WP title, target unit title, its CONTRACT (A->B|P), clarity %.
       Show: progress so far (units_done / units_total).
       IF denied -> mark unit STATUS: ABORTED, output B, STOP.
  4. Execute the single unit-work:
       DEFAULT (before execution, if unit_index > 1):
         Retrieve previous unit''s Result from WP file.
         Retrieve current unit''s A (input state) from WP file.
         Call gem2_bridge(
           source_output=previous_unit_Result,
           target_input=current_unit_A,
           session_context=project_slug + WP title + unit context
         ).
         Read bridge_verdict from response:
           IF bridge_verdict = VALID -> proceed normally. Record bridge_verdict in B.
           IF bridge_verdict = PARTIAL -> warn human: "Inter-unit handoff has gaps: {details}. Proceed anyway?"
             IF human says yes -> proceed. Record bridge_verdict = PARTIAL in B.
             IF human says no -> mark unit STATUS: ABORTED, output B, STOP.
           IF bridge_verdict = INVALID -> warn human: "Previous unit output does not satisfy this unit''s input contract: {details}. Recommend reviewing unit {N-1} result before proceeding."
             IF human says proceed anyway -> proceed. Record bridge_verdict = INVALID in B.
             IF human says no -> mark unit STATUS: ABORTED, output B, STOP.
         IF gem2_bridge call fails -> proceed without validation (bridge_verdict = SKIPPED).
       FALLBACK:
         (no bridge validation -- proceed directly. bridge_verdict = bot)
       IF unit_index = 1 -> skip bridge validation (no previous unit). bridge_verdict = bot.
       Update unit STATUS: PENDING -> IN_PROGRESS in WP file.
       Update WP-level STATUS -> IN_PROGRESS in WP header.
       Read unit''s CONTRACT (A, B, P) -> fulfill B given A under P.
       Execution strategy is executor''s choice (blackbox).
       On completion:
         Update unit STATUS -> COMPLETED, write Result in WP file.
         Refine Tags if implementation diverged from plan:
           add tags for discovered concerns, keep existing valid tags.
           (* Tag format: /^[a-z]+-[a-z]+(-[a-z]+)*$/ -- {verb-ing}-{object} *)
  5. Verify the completed unit (inline):
       Invoke /verify-work(wp_path, unit_index=N) -- SINGLE mode.
       Read back unit_state from verify-work output.
       IF unit_state = SUCCESS:
         Set retried = bot. Proceed to step 6.
       IF unit_state = FAILURE:
         Branch on execution mode:
           INTERACTIVE (human present):
             Report FAILURE with detail from verify-work.
             Ask human: retry (same CONTRACT, different strategy) / skip (accept FAILURE) / abort unit.
             IF retry -> clear Result field, re-execute step 4, re-verify. Set retried = T.
             IF skip -> accept FAILURE state, set retried = bot. Proceed to step 6.
             IF abort -> mark unit STATUS: ABORTED, output B, STOP.
           AUTONOMOUS (human said "all units autonomously"):
             Retry once: clear Result field, re-execute step 4 with different strategy, re-verify.
             Set retried = T.
             IF retry succeeds (STATE = SUCCESS) -> proceed to step 6.
             IF retry also FAILURE -> report to human regardless of mode.
               Human decides: skip / abort / manual intervention.
  6. Finalize:
       DEFAULT:
         Derive wp_status from all unit statuses in WP file.
         Call gem2_task_update(
           task_id=WP.task_id,
           status=wp_status mapped to task status,
           result_summary=output_summary
         ).
         IF wp_status = COMPLETED -> gem2_task_update(status="COMPLETED").
         IF wp_status = ABORTED -> gem2_task_update(status="ABORTED").
         IF wp_status = IN_PROGRESS -> gem2_task_update(status="IN_PROGRESS", result_summary="Unit {N} done").
         IF gem2_task_update fails -> continue (WP file is source of truth).
       FALLBACK:
         (no remote task sync)
       Record updated_at timestamp. Output B.
>>

(* === Constraint === *)
CONSTRAINT := [
  |- ONE unit-work per invocation -- never batch multiple units,
  |- NEVER proceed without human permission -- always ask first,
  |- NEVER write State field directly -- invoke /verify-work for all verification,
  |- NEVER archive -- that is /archive-work mandate,
  |- NEVER plan or decompose -- that is /plan-work mandate,
  |- NEVER modify CONTRACTs -- work against what was planned. Retry uses same CONTRACT, different execution strategy,
  |- Execution strategy is executor''s blackbox -- skill does not dictate how,
  |- Maximum ONE retry per unit -- after retry FAILURE, human must decide,
  |- gem2-lfs/gem2-studio unavailability does NOT block execution -- WP file is source of truth,
  |- Bridge validation warns but does NOT auto-block -- human decides on PARTIAL/INVALID verdicts
]

(* === Invariant === *)
INV := [
  |- Per-unit STATUS in WP file is the source of truth for unit progress,
  |- Unit STATUS transitions: PENDING -> IN_PROGRESS -> COMPLETED or ABORTED,
  |- Every COMPLETED unit has been verified before proceeding to the next unit,
  |- WP-level STATUS derived: any IN_PROGRESS -> IN_PROGRESS, all COMPLETED -> COMPLETED,
  |- Result field filled in WP file after each unit completes,
  |- State field written by /verify-work (SINGLE mode) -- never by proceed-work directly,
  |- Tags may be refined on completion -- add implementation-discovered tags, keep valid plan-time tags,
  |- Human permission required before execution -- no silent work,
  |- CONTRACTs are read-only during execution -- retry is a new attempt, not a scope change,
  |- WP file is ALWAYS updated (L0 baseline) -- gem2-lfs sync is best-effort (DEFAULT),
  |- DEFAULT: gem2_bridge validates inter-unit handoff for unit_index > 1 -- FALLBACK skips,
  |- DEFAULT: gem2_task_update syncs status to gem2-lfs -- FALLBACK skips,
  |- MANDATE BOUNDARY: proceed-work executes and records results -- it does NOT plan, verify-state, or archive
]

(* === Pre-Execution Dialog === *)
Ask_Human := <<
  [field: "confirm",
   prompt: "Proceed with unit-work {N}: {title}? (shows CONTRACT + progress)",
   required: T]
>>

(* === Post-Execution Routing === *)
(* G_ij bridges -- formal handoff contracts *)
Routing := [
  G(proceed-work, proceed-work):
    wp_status = IN_PROGRESS -> B_proceed.wp_path feeds A_proceed.wp_path (next PENDING unit),
  G(proceed-work, archive-work):
    wp_status = COMPLETED -> B_proceed.wp_path feeds A_archive.wp_path,
  G(proceed-work, archive-work):
    wp_status = ABORTED -> B_proceed.wp_path feeds A_archive.wp_path
]
',
  '["core-skill", "proceed-work", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-search-kg',
  'gem2-studio',
  'prompt',
  'search-kg SKILL.md',
  '---
name: search-kg
description: >
  (Who) Any role needing prior art — typically ARCHITECT before /plan-work.
  (What) Search proven unit-contracts and live work-plans for reusable patterns via
  tag-based filesystem search (L0) and FTS + semantic similarity (L1 DEFAULT).
  (When) Before /plan-work decomposition, or anytime human asks "have we done something like this before?"
  (Where) .gem-squared/archive/ + .gem-squared/work-plan/ (local filesystem) and gem2-lfs knowledge store (DEFAULT).
  (Why) Proven CONTRACTs are the reusable unit of TPMN. Without retrieval, the archive is a graveyard, not a library.
  L1 adds cross-project semantic search and FTS over persisted knowledge.
argument-hint: "[search query — what pattern are you looking for?]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Glob
  - Grep
  - Bash(date *)
  - gem2_knowledge_search
  - gem2_semantic_search
---

(* TPMN SKILL — search-kg (L1) *)

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "Any role needing prior art — typically ARCHITECT before /plan-work",
  what:  "Search proven unit-contracts and live work-plans for reusable patterns.
          DEFAULT: FTS + semantic search via gem2-lfs, merged with local filesystem results.
          FALLBACK: tag-based Glob/Grep/Read on archive/ + work-plan/ (pure L0)",
  when:  "Before /plan-work decomposition, or anytime human asks ''have we done something like this before?''",
  where: "DEFAULT: gem2-lfs knowledge store + .gem-squared/archive/ + .gem-squared/work-plan/.
          FALLBACK: .gem-squared/archive/ + .gem-squared/work-plan/ (local filesystem only)",
  why:   "Proven CONTRACTs are the reusable unit of TPMN. Without retrieval, the archive is a graveyard, not a library.
          L1 adds FTS and vector similarity for richer recall and cross-project pattern discovery"
]

(* === Input === *)
A ≜ [
  query: 𝕊,                            (* what the human is looking for — natural language *)
  project_slug: 𝕊,
  scope: {archive, live, all}?,         (* ⊥ = all. archive = proven only, live = work-plan/ only *)
  state_filter: {SUCCESS, FAILURE, —}?, (* ⊥ = no filter. SUCCESS = proven contracts only *)
  limit: ℕ?,                            (* ⊥ = 5. max results to return *)
  layer: {DEFAULT, FALLBACK}            (* inherited from /init-session or runtime detection *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  searched_at: 𝕊,                      (* ISO8601 *)
  query: 𝕊,                            (* echoed back *)
  results: Seq([
    wp_id: 𝕊,                          (* e.g., "WP-ST-69" *)
    wp_title: 𝕊,                       (* WP-level title *)
    wp_state: {SUCCESS, FAILURE, —},   (* WP-level STATE *)
    source: {archive, live, remote},   (* which origin — remote = gem2-lfs only *)
    unit_index: ℕ,                      (* 1-based unit number within WP *)
    unit_title: 𝕊,                     (* unit-work title *)
    unit_state: {SUCCESS, FAILURE, —}, (* per-unit State *)
    contract: [                         (* extracted CONTRACT *)
      a: 𝕊,                            (* input specification *)
      b: 𝕊,                            (* output specification *)
      p: 𝕊                             (* preconditions *)
    ],
    result_summary: 𝕊?,                (* Result field — ⊥ if PENDING *)
    relevance: {exact, partial, weak},  (* keyword match quality *)
    similarity_score: ℝ?               (* ⊥ for L0/filesystem results. ℝ[0,1] for semantic matches *)
  ]),
  result_count: ℕ,                      (* |results| *)
  sources_searched: Seq(𝕊),            (* directories + remote sources queried *)
  layer_used: {DEFAULT, FALLBACK}       (* which layer actually executed *)
]

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER modifies any file — zero side effects on filesystem,
  ⊢ NEVER executes work — pattern retrieval only,
  ⊢ NEVER plans work — that is /plan-work mandate,
  ⊢ NEVER ranks by recency alone — relevance to query is primary sort key,
  ⊢ NEVER returns results without extracted CONTRACT (A, B, P) — titles alone are insufficient,
  ⊢ NEVER calls gem2_truth_filter — that is L2 only
]

(* === Precondition === *)
P ≜ query ≠ ⊥
    ∧ project_slug ≠ ⊥
    ∧ (".gem-squared/archive/" exists ∨ ".gem-squared/work-plan/" exists)

(* === Layers === *)
L0 ≜ "git + .gem-squared/ files — tag-based Glob/Grep search on archive/ + work-plan/"
L1 ≜ "gem2-lfs — additionally FTS via gem2_knowledge_search and vector similarity via gem2_semantic_search.
       Merges remote results with local filesystem results. Cross-project patterns available"

(* === Transform === *)
F ≜ <<
  1. Tag-based filesystem search (ALWAYS — both DEFAULT and FALLBACK):
       Determine scope directories:
         scope = archive → .gem-squared/archive/ only.
         scope = live    → .gem-squared/work-plan/ only.
         scope = all     → both directories.
       Glob *.md in scope directories → candidate WP files.
       Grep for `- Tags:` lines across all candidate files → tag index.
       Derive search tags from query:
         Convert query to {verb-ing}-{object} candidates.
         e.g., "terminal kill" → [killing-terminal, closing-terminal, managing-terminal].
       For each WP file with tag matches:
         Read header → extract wp_id, wp_title, wp_state.
         IF state_filter ≠ ⊥ ∧ wp_state ≠ state_filter → skip.
         Parse unit-works with matching tags:
           For each ### {N}. {title} | STATUS: {status} section:
             Extract Tags:, A:, B:, P:, Result:, State: fields.
             Score relevance:
               exact:   query tag matches a unit tag fully,
               partial: query verb OR object matches a tag component,
               weak:    query words found in unit title or A/B/P (fallback).
       Collect local_results with source = archive | live.

  2. Remote knowledge search (DEFAULT only — skip on FALLBACK):
       DEFAULT:
         Call gem2_knowledge_search(project_slug, query, tags=derived_search_tags).
           → FTS-based search across persisted knowledge entries.
           → Returns knowledge documents with wp_id, contract, tags metadata.
         Call gem2_semantic_search(query, project_slug).
           → Vector similarity over embedded knowledge documents.
           → Returns scored results with similarity_score ∈ ℝ[0,1].
         Parse remote results: extract wp_id, unit_index, contract fields from knowledge metadata.
         Remote results may include cross-project patterns (project_slug filter is optional in semantic).
         Merge remote_results with local_results:
           Deduplicate by (wp_id, unit_index) — local wins on conflict.
           Remote-only results get source = remote.
           Attach similarity_score from semantic search where available.
       FALLBACK (gem2-lfs unreachable):
         Skip. local_results from step 1 are the complete result set.
         Set layer_used = FALLBACK.

  3. Output B:
       Sort merged results: archive before live before remote,
         SUCCESS before FAILURE, exact before partial before weak.
         Within same relevance tier: higher similarity_score first (if available).
       Trim to limit.
       Record searched_at timestamp.
       Assemble results Seq with all fields populated.
       Record sources_searched (directories + "gem2-lfs" if DEFAULT succeeded).
       Set layer_used = DEFAULT | FALLBACK.
       Output B as structured report — human-readable table + raw data.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ Strictly read-only — NEVER modifies any file,
  ⊢ NEVER executes work — pattern retrieval only,
  ⊢ NEVER plans work — that is /plan-work mandate,
  ⊢ NEVER ranks by recency alone — relevance to query is primary sort key,
  ⊢ gem2-lfs failure does NOT block search — FALLBACK to filesystem results is always available,
  ⊢ NEVER references gem2_truth_filter — that is L2 only
]

(* === Invariant === *)
INV ≜ [
  ⊢ Zero side effects — filesystem unchanged after execution,
  ⊢ B is state (search results), not action — AI reads B and decides what to do with patterns,
  ⊢ Every result includes the full extracted CONTRACT (A, B, P) — not just titles,
  ⊢ Source provenance is always recorded: archive vs live vs remote, SUCCESS vs FAILURE,
  ⊢ Output is self-sufficient — human and AI can reuse patterns from search alone,
  ⊢ Proven contracts (archive + SUCCESS) ranked above live/unverified contracts,
  ⊢ DEFAULT adds remote results but never removes local results — L1 is additive over L0,
  ⊢ FALLBACK produces identical output to L0 — no degradation in filesystem search quality,
  ⊢ This skill is the READ counterpart to /archive-work''s WRITE of proven contracts,
  ⊢ MANDATE BOUNDARY: pattern retrieval only — never execution, planning, or modification
]

(* === Post-Execution Routing === *)
(* G_ij bridges — formal handoff contracts *)
Routing ≜ [
  G(search-kg, plan-work):
    result_count > 0 → B_search.results feeds A_plan.references_used,
  G(search-kg, plan-work):
    result_count = 0 → no prior art, A_plan.work proceeds from scratch,
  G(search-kg, ⊥):
    human_browsing → display results, await human decision
]
',
  '["core-skill", "search-kg", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-search-skill',
  'gem2-studio',
  'prompt',
  'search-skill SKILL.md',
  '---
name: search-skill
description: >
  (Who) Any role needing domain capabilities — typically AI Pilot during /plan-work.
  (What) Search .claude/skills/ and .gem-squared/external-skills/ for installed and archived
  skills relevant to the current task via filesystem discovery (L0) and gem2-lfs skill store +
  semantic similarity (L1 DEFAULT).
  (When) During /plan-work when domain capability is needed, or anytime human asks "what skills do we have?"
  (Why) Market-standard skills in .claude/skills/ contain domain knowledge. Archived skills in
  .gem-squared/external-skills/ remain discoverable. gem2-lfs extends discovery across projects.
  L1 adds cross-project skill search and vector similarity over skill descriptions.
argument-hint: "[search query — what capability are you looking for?]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Glob
  - Grep
  - Bash(date *)
  - gem2_skill_search
  - gem2_semantic_search
---

(* TPMN SKILL — search-skill (L1) *)

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "Any role needing domain capabilities — typically AI Pilot during /plan-work",
  what:  "Search .claude/skills/ and .gem-squared/external-skills/ for installed and archived skills.
          DEFAULT: filesystem discovery merged with gem2-lfs skill store + semantic similarity.
          FALLBACK: Glob/Read on .claude/skills/*/ and .gem-squared/external-skills/*/ (pure L0)",
  when:  "During /plan-work for domain capability discovery, or anytime human asks ''what skills do we have?''",
  where: "DEFAULT: gem2-lfs skill store + .claude/skills/*/ + .gem-squared/external-skills/*/.
          FALLBACK: .claude/skills/*/ + .gem-squared/external-skills/*/ (local filesystem only)",
  why:   "Market-standard skills encode domain knowledge (Figma, Sentry, deploy, etc.). Archived skills remain
          discoverable as CONTRACTs. TPMN orchestrates them; this skill discovers them.
          L1 adds cross-project skill search and vector similarity for richer recall"
]

(* === Input === *)
A ≜ [
  query: 𝕊?,                            (* ⊥ = list all installed skills. 𝕊 = search by keyword *)
  project_slug: 𝕊,
  include_tpmn: 𝔹?,                     (* ⊥ = true. Include TPMN-extracted skills in results *)
  layer: {DEFAULT, FALLBACK}             (* inherited from /init-session or runtime detection *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  searched_at: 𝕊,                      (* ISO8601 *)
  query: 𝕊?,                           (* echoed back — ⊥ if catalog listing *)
  skills: Seq([
    skill_name: 𝕊,                     (* from YAML name field, or dirname if no SKILL.md *)
    skill_path: Path,                   (* e.g., ".claude/skills/figma-to-code/SKILL.md" or ".claude/skills/agents/" *)
    description: 𝕊,                    (* from YAML description field — first 200 chars. "Directory without SKILL.md" if none *)
    format: {TPMN, prose, directory, unknown},  (* directory = no SKILL.md present *)
    has_scripts: 𝔹,                    (* scripts/ subdirectory exists *)
    has_references: 𝔹,                 (* references/ subdirectory exists *)
    relevance: {exact, partial, none},  (* keyword match against query *)
    location: {active, archived, remote},  (* active = .claude/skills/, archived = .gem-squared/external-skills/, remote = gem2-lfs only *)
    similarity_score: ℝ?               (* ⊥ for L0/filesystem results. ℝ[0,1] for semantic matches *)
  ]),
  skill_count: ℕ,                       (* total items found (local + remote, deduplicated) *)
  tpmn_count: ℕ,                        (* skills in TPMN format *)
  prose_count: ℕ,                       (* skills in prose/market-standard format *)
  directory_count: ℕ,                   (* directories without SKILL.md *)
  match_count: ℕ,                       (* items matching query — 0 if catalog listing *)
  archived_count: ℕ,                    (* items in .gem-squared/external-skills/ *)
  remote_skills: Seq([                  (* skills found in gem2-lfs but NOT on local filesystem — DEFAULT only *)
    skill_name: 𝕊,
    description: 𝕊,
    source_project: 𝕊,                 (* project_slug where skill is registered *)
    similarity_score: ℝ?               (* ⊥ for FTS results. ℝ[0,1] for semantic matches *)
  ]),
  remote_count: ℕ,                      (* total remote-only results — 0 on FALLBACK *)
  layer_used: {DEFAULT, FALLBACK}       (* which layer actually executed *)
]

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER modifies any file — zero side effects on .claude/skills/ and .gem-squared/external-skills/,
  ⊢ NEVER converts or reformats skills — reads all formats AS-IS,
  ⊢ NEVER executes discovered skills — discovery and catalog only,
  ⊢ NEVER searches .gem-squared/archive/ — that is /search-kg mandate,
  ⊢ NEVER ranks prose vs TPMN by quality — reports format neutrally,
  ⊢ NEVER calls gem2_truth_filter — that is L2 only
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ "{project_root}/.claude/skills/" exists

(* === Layers === *)
L0 ≜ "git + .gem-squared/ files — Glob/Read on .claude/skills/*/ and .gem-squared/external-skills/*/. Pure filesystem discovery"
L1 ≜ "gem2-lfs — additionally gem2_skill_search for registered skills across projects and
       gem2_semantic_search for vector similarity over skill descriptions/content.
       Merges remote results with local filesystem results. Cross-project skills available"

(* === Transform === *)
F ≜ <<
  1. Discover ALL items from active and archived locations (ALWAYS — both DEFAULT and FALLBACK):
       Scope 1 (active): Glob {project_root}/.claude/skills/*/ → all active directories.
       Scope 2 (archived): Glob {project_root}/.gem-squared/external-skills/*/ → all archived directories.
       (* Globs ALL subdirectories, not just SKILL.md holders — catches agents/, scripts-only dirs, etc. *)
       FOR each directory in merged list:
         IF SKILL.md exists in directory:
           Parse YAML frontmatter → extract name, description.
           Detect format:
             IF body contains "(* ===" OR "≜" OR "CONSTRAINT ≜" → format = TPMN.
             ELSE IF body contains markdown headers and prose → format = prose.
             ELSE → format = unknown.
         ELSE (no SKILL.md):
           skill_name = dirname.
           description = "Directory without SKILL.md".
           format = directory.
         Check subdirectories:
           has_scripts = scripts/ exists under dir.
           has_references = references/ exists under dir.
         Set location = active or archived based on scope.
         Set similarity_score = ⊥ (filesystem results have no vector score).

  2. Filter by query (IF query ≠ ⊥) (ALWAYS — both DEFAULT and FALLBACK):
       For each item:
         Match query keywords against: name, description, body content (first 500 chars if SKILL.md exists).
         Score relevance:
           exact:   query appears in name or first line of description,
           partial: query words found in description or body,
           none:    no match.
       IF include_tpmn = false → exclude format = TPMN from results.
       Sort: exact before partial. Within same relevance, alphabetical by name.

  3. IF query = ⊥ → catalog listing (ALWAYS — both DEFAULT and FALLBACK):
       Return all items (active + archived), sorted alphabetical.
       All relevance = none (not a search, just discovery).

  4. Compute local counts (ALWAYS — both DEFAULT and FALLBACK):
       skill_count = total items found (active + archived).
       tpmn_count = count where format = TPMN.
       prose_count = count where format = prose.
       directory_count = count where format = directory.
       match_count = count where relevance ∈ {exact, partial}.
       archived_count = count where location = archived.
       Collect local_results.

  5. Remote skill search (DEFAULT only — skip on FALLBACK):
       DEFAULT:
         Call gem2_skill_search(query, limit=20).
           → Search gem2-lfs skill store for registered skills across projects.
           → Returns skill entries with skill_name, description, source_project metadata.
         Call gem2_semantic_search(query, project_slug).
           → Vector similarity over skill descriptions/content.
           → Returns scored results with similarity_score ∈ ℝ[0,1].
         Parse remote results: extract skill_name, description, source_project fields.
         Merge remote_results with local_results:
           Deduplicate by skill_name — local wins on conflict.
           Remote-only results get location = remote.
           Attach similarity_score from semantic search where available.
         Populate B.remote_skills with skills found in gem2-lfs but NOT on local filesystem.
         Set B.remote_count = |remote_skills|.
         Update B.skill_count to include remote-only results.
       FALLBACK (gem2-lfs unreachable):
         Skip. local_results from steps 1–4 are the complete result set.
         Set B.remote_skills = [].
         Set B.remote_count = 0.
         Set B.layer_used = FALLBACK.

  6. Output B as structured report:
       Table format: name | location | format | relevance | similarity | description (truncated).
       IF remote_count > 0 → separate "Remote-only skills" section below local table.
       Summary line: "{skill_count} items ({archived_count} archived, {remote_count} remote, {tpmn_count} TPMN, {prose_count} prose, {directory_count} dirs), {match_count} matching query."
       Record layer_used = DEFAULT | FALLBACK.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ Strictly read-only — NEVER modifies any file in .claude/skills/ or .gem-squared/external-skills/,
  ⊢ NEVER converts or reformats skills — reads market-standard skills AS-IS,
  ⊢ NEVER judges prose vs TPMN — reports format, does not rank by format quality,
  ⊢ NEVER executes skills — discovery only,
  ⊢ Searches .claude/skills/ AND .gem-squared/external-skills/ — does NOT search .gem-squared/archive/ (that is /search-kg mandate),
  ⊢ Discovers ALL subdirectories — not just those with SKILL.md,
  ⊢ gem2-lfs failure does NOT block search — FALLBACK to filesystem results is always available,
  ⊢ NEVER references gem2_truth_filter — that is L2 only
]

(* === Invariant === *)
INV ≜ [
  ⊢ B is state (skill catalog), not action — AI reads B and decides how to use discovered skills,
  ⊢ Zero side effects — .claude/skills/ and .gem-squared/external-skills/ unchanged after execution,
  ⊢ Format detection is heuristic, not authoritative — TPMN markers are sufficient but not exhaustive,
  ⊢ Both TPMN and prose skills are first-class results — no preference in ranking,
  ⊢ Directories without SKILL.md are reported as format=directory — discoverable but not parseable as skills,
  ⊢ This skill is the READ counterpart to /extract-skill''s WRITE to .claude/skills/,
  ⊢ Dual scope: .claude/skills/ (active) + .gem-squared/external-skills/ (archived by /skill-to-kg),
  ⊢ Clear mandate boundary: /search-skill = skills discovery, /search-kg = .gem-squared/archive/ (proven WPs),
  ⊢ DEFAULT adds remote results but never removes local results — L1 is additive over L0,
  ⊢ FALLBACK produces identical output to L0 — no degradation in filesystem search quality,
  ⊢ Remote-only skills (location=remote) are discoverable but not locally installed — human decides next step,
  ⊢ MANDATE BOUNDARY: skill discovery only — never execution, planning, or modification
]

(* === Post-Execution Routing === *)
(* G_ij bridges — formal handoff contracts *)
Routing ≜ [
  G(search-skill, plan-work):
    match_count > 0 → B_search.skills feeds /plan-work''s domain capability context,
  G(search-skill, plan-work):
    match_count = 0 → decompose with TPMN lifecycle only (no domain skills available),
  G(search-skill, ⊥):
    human_browsing → display catalog, await human decision (terminal),
  G(search-skill, ⊥):
    remote_count > 0 → display remote-only skills for potential install decision
]
',
  '["core-skill", "search-skill", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-skill-to-kg',
  'gem2-studio',
  'prompt',
  'skill-to-kg SKILL.md',
  '---
name: skill-to-kg
description: >
  (Who) AI agent (e.g., Claude Code) or human managing skill directory hygiene.
  (What) Archive non-core skills from .claude/skills/ to .gem-squared/external-skills/, or restore/list.
  L1 DEFAULT adds gem2_knowledge_create registration for each archived/restored skill.
  (When) At /init-session (skill hygiene), after extracting project skills, or on user request.
  (Where) .claude/skills/ <-> .gem-squared/external-skills/ (L0 filesystem);
  gem2-lfs via gem2_knowledge_create (L1 DEFAULT).
  (Why) Only 12 core lifecycle skills + project identity belong active. Others become searchable CONTRACTs.
  L1 adds remote knowledge registration so archived/restored events are discoverable via gem2-lfs semantic search.
argument-hint: "[archive (default)|restore <skill-name>|list]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Glob
  - Bash(mv *)
  - Bash(mkdir *)
  - Bash(ls *)
  - Bash(date *)
  - gem2_knowledge_create
---

(* TPMN SKILL — skill-to-kg — L1 *)

(* === Layers === *)
L0 ≜ "Glob .claude/skills/ → filter PROTECTED → mv to .gem-squared/external-skills/ (or reverse for restore, or list).
      Fully local — OS filesystem only. No MCP dependency."
L1 ≜ "DEFAULT: L0 + gem2_knowledge_create (register each archive/restore event as knowledge entity).
      FALLBACK: degrade to L0 when gem2-lfs is unreachable — filesystem move still completed, kg_registered = ⊥"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "Claude Code agent or human managing skill directory hygiene",
  what:  "Archive all non-core, non-project skills from .claude/skills/ to .gem-squared/external-skills/ (or restore/list).
          L1 DEFAULT: additionally register each archive/restore event as a knowledge entity in gem2-lfs",
  when:  "At /init-session (automatic hygiene), after /extract-skill creates a project skill, or on explicit user request",
  where: "L0: {project_root}/.claude/skills/ <-> {project_root}/.gem-squared/external-skills/ — local filesystem only.
          L1 DEFAULT: additionally gem2-lfs via gem2_knowledge_create",
  why:   "In TPMN mode, .claude/skills/ should contain only PROTECTED items (12 core + project identity). Non-core skills become CONTRACTs — searchable via /search-skill, not active triggers.
          L1 adds: remote knowledge registration so archive/restore events are discoverable across sessions via gem2-lfs semantic search"
]

(* === Protected — never archived === *)
CORE_SKILLS ≜ {
  "archive-work", "check-session", "end-session", "extract-skill",
  "init-session", "plan-work", "proceed-work", "search-kg",
  "search-skill", "skill-to-kg", "update-work-plan", "verify-work"
}  (* 12 official TPMN lifecycle skills *)

PROJECT_SKILL ≜ {project_slug}
(* The project identity skill bound by CLAUDE.md — always protected alongside CORE_SKILLS *)

PROTECTED ≜ CORE_SKILLS ∪ {PROJECT_SKILL}

(* === Input === *)
A ≜ [
  operation: {archive, restore, list}?,  (* ⊥ = archive (default). archive = batch move ALL non-core out *)
  skill_name: 𝕊?,                     (* required for restore only. ⊥ for archive/list *)
  project_slug: 𝕊
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  operation: {archive, restore, list},
  moved_skills: Seq(𝕊)?,              (* names of skills/dirs moved out — archive only *)
  moved_count: ℕ?,                     (* |moved_skills| — archive only. 0 = nothing to archive *)
  restored_skill: 𝕊?,                 (* name restored — restore only *)
  source_path: Path?,                  (* restore: moved FROM *)
  dest_path: Path?,                    (* restore: moved TO *)
  archived_skills: Seq([               (* list: contents of external-skills/ *)
    name: 𝕊,
    path: Path,
    has_skill_md: 𝔹
  ])?,
  active_non_core: Seq(𝕊)?,           (* list: non-core items still in .claude/skills/ *)
  layer: {DEFAULT, FALLBACK},          (* which execution path was taken *)
  kg_registered: 𝔹?,                  (* L1: ⊤ if gem2_knowledge_create succeeded, ⊥ if FALLBACK *)
  completed_at: 𝕊                     (* ISO8601 *)
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ (operation = restore ⟹ skill_name ≠ ⊥)
    ∧ (operation = restore ⟹ ".gem-squared/external-skills/{skill_name}/" exists)

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER moves PROTECTED skills (12 core + project identity) — permanently active,
  ⊢ NEVER deletes skills — bidirectional move only, fully reversible,
  ⊢ NEVER modifies skill content — moves directory as-is,
  ⊢ NEVER overwrites existing directory at destination — skips with conflict log,
  ⊢ NEVER touches ~/.claude/skills/ (global) — project-local scope only,
  ⊢ NEVER blocks on gem2-lfs failure — degrade to FALLBACK silently (filesystem move still completed),
  ⊢ NEVER calls gem2_truth_filter — that is L2 only
]

(* === Transform === *)
F ≜ <<
  0. Resolve default operation:                                  (ALWAYS — both DEFAULT and FALLBACK)
       IF operation = ⊥ → operation ≜ archive.
       (* No args = archive. User must explicitly say "restore <name>" or "list". *)

  1. Ensure archive directory exists:                            (ALWAYS — both DEFAULT and FALLBACK)
       mkdir -p {project_root}/.gem-squared/external-skills/

  2. Execute operation:                                          (ALWAYS — both DEFAULT and FALLBACK)
       IF operation = archive:
         Glob {project_root}/.claude/skills/*/ → all_dirs.
         (* Globs ALL subdirectories, not just SKILL.md holders — catches agents/, etc. *)
         candidates ≜ [d | d ∈ all_dirs, d.name ∉ PROTECTED]
         IF |candidates| = 0 → skip (nothing to archive), output B with moved_count=0.
         ELSE →
           (* MANDATORY USER NOTIFICATION — print BEFORE moving anything *)
           Print to user:
             "TPMN Skill Hygiene — /skill-to-kg archive
              Moving {|candidates|} non-core item(s) from .claude/skills/ to .gem-squared/external-skills/:
                {list each candidate name}
              - NOT deleted — moved to .gem-squared/external-skills/
              - Still searchable via /search-skill and /search-kg
              - Restore anytime: /skill-to-kg restore <name>"
           moved_skills ≜ []
           FOR each dir in candidates:
             IF .gem-squared/external-skills/{dir.name}/ exists → skip, log conflict.
             ELSE → mv .claude/skills/{dir.name}/ → .gem-squared/external-skills/{dir.name}/
             Append dir.name to moved_skills.
           moved_count ≜ |moved_skills|.

       IF operation = restore:
         Verify .gem-squared/external-skills/{skill_name}/ exists.
         IF .claude/skills/{skill_name}/ already exists → STOP, report conflict.
         mv .gem-squared/external-skills/{skill_name}/ → .claude/skills/{skill_name}/
         Record restored_skill, source_path, dest_path.

       IF operation = list:
         Glob .gem-squared/external-skills/*/ → archived_skills (check has_skill_md per dir).
         Glob .claude/skills/*/ → all_active.
         active_non_core ≜ [d.name | d ∈ all_active, d.name ∉ PROTECTED].

  3. Record completed_at timestamp.                              (ALWAYS — both DEFAULT and FALLBACK)

     (* --- L1 layer switch: register knowledge --- *)
       DEFAULT (gem2-lfs reachable):

         IF operation = archive ∧ moved_count > 0:
           FOR each skill_name in moved_skills:
             Call gem2_knowledge_create(
               entity_type   = "skill-archive",
               title         = "Archived: {skill_name}",
               content       = "Skill archived from .claude/skills/ to .gem-squared/external-skills/",
               project_slug  = project_slug,
               tags          = ["skill-archive", skill_name]
             ).
           Set B.kg_registered = ⊤.

         IF operation = restore:
           Call gem2_knowledge_create(
             entity_type   = "skill-restore",
             title         = "Restored: {skill_name}",
             content       = "Skill restored from .gem-squared/external-skills/ to .claude/skills/",
             project_slug  = project_slug,
             tags          = ["skill-restore", skill_name]
           ).
           Set B.kg_registered = ⊤.

         IF operation = list:
           (* No MCP calls needed for list — read-only operation *)
           Set B.kg_registered = ⊥.

         Set B.layer = DEFAULT.

       FALLBACK (gem2-lfs unreachable):
         B.kg_registered = ⊥.
         Set B.layer = FALLBACK.
         (* Filesystem move already completed in step 2 — L0 baseline is fully functional *)

     Output B.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ NEVER move PROTECTED (CORE_SKILLS ∪ {PROJECT_SKILL}) — permanently active,
  ⊢ NEVER delete skills — move only (bidirectional, reversible),
  ⊢ NEVER modify skill content — move directory as-is,
  ⊢ NEVER overwrite existing directory at destination — skip with conflict log,
  ⊢ NEVER touch ~/.claude/skills/ (global) — project-local {project_root}/.claude/skills/ only,
  ⊢ archive is BATCH — all non-protected directories moved in one invocation, not one by one,
  ⊢ Default operation is archive — no args means archive, not list or restore,
  ⊢ ALWAYS notify user before archive — list skill names, explain they are NOT deleted, show restore command,
  ⊢ gem2-lfs failure does NOT block filesystem operations — move completes regardless,
  ⊢ gem2_knowledge_create is called per-skill for archive (one entity per archived skill), once for restore
]

(* === Invariant === *)
INV ≜ [
  ⊢ After archive: {project_root}/.claude/skills/ contains ONLY PROTECTED items — zero non-protected,
  ⊢ A skill/dir exists in EITHER .claude/skills/ OR .gem-squared/external-skills/, never both,
  ⊢ Archived items retain full directory structure (SKILL.md, references/, scripts/, etc.),
  ⊢ /search-skill finds both active AND archived skills (with location field),
  ⊢ Archived skills are CONTRACTs — searchable via /search-kg, not active triggers,
  ⊢ CORE_SKILLS (12) are never archivable — hard-coded protection,
  ⊢ PROJECT_SKILL ({project_slug}) is never archivable — project identity protection,
  ⊢ Non-SKILL.md directories (e.g., agents/) are archived like any other non-protected item,
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ B.kg_registered = ⊤ only when gem2_knowledge_create succeeded (DEFAULT path, archive or restore),
  ⊢ B.kg_registered = ⊥ for list operations (no knowledge registration needed) and FALLBACK,
  ⊢ DEFAULT: filesystem move + gem2_knowledge_create per archived/restored skill,
  ⊢ FALLBACK: filesystem move only — fully functional without gem2-lfs,
  ⊢ MANDATE BOUNDARY: skill location management only — never skill content modification
]

(* === Post-Execution Routing === *)
(* G_ij bridges — formal handoff contracts *)
Routing ≜ [
  G(skill-to-kg, ⊥):
    operation = archive ∧ layer = DEFAULT  → report "{moved_count} non-protected items archived, registered in gem2-lfs" (terminal),
  G(skill-to-kg, ⊥):
    operation = archive ∧ layer = FALLBACK → report "{moved_count} non-protected items archived (FALLBACK — no kg registration)" (terminal),
  G(skill-to-kg, ⊥):
    operation = restore ∧ layer = DEFAULT  → report "{skill_name} restored to .claude/skills/, registered in gem2-lfs" (terminal),
  G(skill-to-kg, ⊥):
    operation = restore ∧ layer = FALLBACK → report "{skill_name} restored to .claude/skills/ (FALLBACK — no kg registration)" (terminal),
  G(skill-to-kg, ⊥):
    operation = list → display catalog (terminal)
]
',
  '["core-skill", "skill-to-kg", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-update-work-plan',
  'gem2-studio',
  'prompt',
  'update-work-plan SKILL.md',
  '---
name: update-work-plan
description: >
  (Who) Human or agent discovering mid-execution scope changes.
  (What) Mutate PENDING unit-works in a live WP — add new units, modify CONTRACTs,
  reorder, or abort. L1 DEFAULT syncs mutations to gem2-lfs task store.
  (When) During /proceed-work execution when scope changes, new requirements emerge,
  or a CONTRACT needs revision.
  (Where) .gem-squared/work-plan/{WP}.md (local filesystem, always written) and
  gem2-lfs task store (DEFAULT, synced after local mutation).
  (Why) Plans are mutable before execution. This is the sanctioned mutation path —
  prevents ad-hoc WP editing that bypasses CONTRACT discipline.
  L1 adds remote task state sync for cross-session visibility.
argument-hint: "[WP path or title] [operation: add|modify|abort|reorder]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash(date *)
  - gem2_task_update
---

(* TPMN SKILL — update-work-plan (L1) *)

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "Human or agent discovering mid-execution scope changes",
  what:  "Mutate PENDING unit-works in a live WP — add new units, modify CONTRACTs, reorder, or abort.
          DEFAULT: additionally sync task status to gem2-lfs after WP file mutation.
          FALLBACK: WP file is the sole state store (pure L0)",
  when:  "During /proceed-work execution when scope changes, new requirements emerge, or a CONTRACT needs revision",
  where: "DEFAULT: .gem-squared/work-plan/{WP}.md (always) + gem2-lfs task store (sync after mutation).
          FALLBACK: .gem-squared/work-plan/{WP}.md (local filesystem only)",
  why:   "Plans are mutable before execution. This is the sanctioned mutation path — prevents ad-hoc WP editing
          that bypasses CONTRACT discipline. L1 adds remote sync so cross-session agents see updated state"
]

(* === Input === *)
A ≜ [
  project_slug: 𝕊,
  wp_path: Path?,                      (* direct path to WP file, or ⊥ *)
  work_title: 𝕊?,                     (* search term if wp_path not given *)
  operation: {add, modify, abort, reorder},
  target_units: Seq(ℕ)?,              (* 1-based unit indices — ⊥ = ask human *)
  description: 𝕊?,                    (* what to change — ⊥ = ask human *)
  layer: {DEFAULT, FALLBACK}           (* inherited from /init-session or runtime detection *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  wp_path: Path,
  wp_id: 𝕊,
  operation: {add, modify, abort, reorder},
  units_affected: Seq(ℕ),             (* which units were changed, 1-based *)
  units_total: ℕ,                      (* new total after mutation *)
  units_pending: ℕ,                    (* remaining PENDING units *)
  change_summary: 𝕊,                  (* human-readable description of what changed *)
  updated_at: 𝕊,                      (* ISO8601 *)
  remote_synced: 𝔹?                   (* ⊤ if gem2-lfs sync succeeded, ⊥ if failed or FALLBACK *)
]

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER executes work — mutation only, execution is /proceed-work mandate,
  ⊢ NEVER verifies — that is /verify-work mandate,
  ⊢ NEVER archives — that is /archive-work mandate,
  ⊢ NEVER creates new WPs — that is /plan-work mandate,
  ⊢ NEVER touches COMPLETED units — their Results are recorded facts,
  ⊢ NEVER touches IN_PROGRESS units — the executor owns them,
  ⊢ NEVER exceeds 9 units per WP — Miller''s law ceiling,
  ⊢ NEVER calls gem2_truth_filter — that is L2 only
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ (wp_path ≠ ⊥ ∨ work_title ≠ ⊥)
    ∧ ".gem-squared/work-plan/" exists
    ∧ WP.STATUS ∈ {PENDING, IN_PROGRESS}   (* not archived, not completed *)

(* === Layers === *)
L0 ≜ "git + .gem-squared/ files — WP file mutation only. No remote sync"
L1 ≜ "gem2-lfs — additionally syncs task status via gem2_task_update after WP file mutation.
       Abort operations update task status to ABORTED in gem2-lfs.
       WP file remains the source of truth; remote is best-effort sync"

(* === Transform === *)
F ≜ <<
  1. Identify WP and validate state:
       IF wp_path provided → read that WP.
       IF only work_title → search work-plan/ for matching WP.
       Parse all units: index, title, STATUS, CONTRACT.
       Validate: WP is not archived (must be in work-plan/, not archive/).
       Validate: at least one PENDING unit exists (for modify/abort/reorder).
       Show current WP state: unit list with STATUS indicators.

  2. Ask human permission and determine scope:
       Show: WP title, operation requested, current unit list.
       IF operation = add:
         Ask: what new unit-work(s) to add (title, A, B, P, Clarity).
         Validate: total units after add ≤ 9 (Miller''s law).
       IF operation = modify:
         Show PENDING units only (COMPLETED/IN_PROGRESS are immutable).
         Ask: which unit(s) to modify, what changes to CONTRACT.
       IF operation = abort:
         Show PENDING units only.
         Ask: which unit(s) to abort and reason.
       IF operation = reorder:
         Show PENDING units only.
         Ask: new order for PENDING units.
       IF denied → STOP, output B with units_affected = [].

  3. Execute mutation on WP file (ALWAYS — both DEFAULT and FALLBACK):
       IF operation = add:
         Append new unit-work(s) to ## Unit-Works section.
         Each new unit gets STATUS: PENDING, empty Result/State/Truth.
         CONTRACT format matches /plan-work output (A, B, P, Clarity, Unclear, Tags).
       IF operation = modify:
         Update CONTRACT fields (A, B, P, Clarity, Unclear, Tags) for target units.
         Tags: add, remove, or replace tags on PENDING units.
           (* Tag format: /^[a-z]+-[a-z]+(-[a-z]+)*$/ — {verb-ing}-{object} *)
         Preserve STATUS: PENDING — do not change status.
         Do NOT touch Result/State/Truth fields (empty for PENDING).
       IF operation = abort:
         Mark target units STATUS: PENDING → ABORTED.
         Write `- Result: Aborted by /update-work-plan ({reason}).`
         Leave State and Truth empty.
       IF operation = reorder:
         Renumber PENDING units to new order.
         COMPLETED/IN_PROGRESS units retain their original numbers.
       Write updated WP file.

  4. Sync to gem2-lfs (DEFAULT only — skip on FALLBACK):
       DEFAULT:
         Extract task_id from WP header.
         Derive updated tags from all non-ABORTED units'' Tags fields.
         Call gem2_task_update(task_id, status=derived_wp_status, tags=derived_tags).
           derived_wp_status:
             all units PENDING → PENDING.
             any unit IN_PROGRESS → IN_PROGRESS.
             all units COMPLETED|ABORTED → COMPLETED.
         IF operation = abort:
           Call gem2_task_update(task_id, status="ABORTED", tags=derived_tags)
             when all remaining units are ABORTED.
         Set remote_synced = ⊤ on success, ⊥ on failure.
         (* gem2-lfs sync failure does NOT block — WP file is the source of truth *)
       FALLBACK (gem2-lfs unreachable):
         Skip remote sync. Set remote_synced = ⊥.

  5. Output B:
       Record updated_at timestamp.
       Assemble B with all fields populated.
       Output B as structured report.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ NEVER execute work — mutation only, execution is /proceed-work mandate,
  ⊢ NEVER verify — that is /verify-work mandate,
  ⊢ NEVER archive — that is /archive-work mandate,
  ⊢ NEVER create new WPs — that is /plan-work mandate,
  ⊢ NEVER touch COMPLETED units — their Results are recorded facts,
  ⊢ NEVER touch IN_PROGRESS units — the executor owns them,
  ⊢ ONLY mutate PENDING units — the unexecuted plan is mutable,
  ⊢ NEVER exceed 9 units per WP — Miller''s law ceiling,
  ⊢ NEVER mutate without human permission — always ask first,
  ⊢ gem2-lfs sync failure does NOT block mutation — WP file is the source of truth,
  ⊢ NEVER reference gem2_truth_filter — that is L2 only
]

(* === Invariant === *)
INV ≜ [
  ⊢ PENDING units are mutable — they are plans, not facts,
  ⊢ COMPLETED units are immutable — their Result/State/Truth are recorded facts,
  ⊢ IN_PROGRESS units are immutable — the executor owns them,
  ⊢ WP-level STATUS unchanged by this skill — derived from unit statuses by other skills,
  ⊢ Every added unit has a CONTRACT (A → B | P) — no uncontracted work,
  ⊢ Every added unit has a Clarity % — no unassessed scope,
  ⊢ Every added unit has 1-3 Tags in {verb-ing}-{object} format — searchable by /search-kg,
  ⊢ Tags on PENDING units are mutable — tags on COMPLETED/IN_PROGRESS units are immutable,
  ⊢ Total units ≤ 9 after mutation — Miller''s law ceiling,
  ⊢ WP file is ALWAYS updated first — gem2-lfs sync is DEFAULT (best-effort on failure),
  ⊢ DEFAULT adds remote sync but never skips local file mutation — L1 is additive over L0,
  ⊢ FALLBACK produces identical WP file output to L0 — no degradation in mutation quality,
  ⊢ MANDATE BOUNDARY: plan mutation only — between /plan-work (creates) and /proceed-work (executes)
]

(* === Pre-Execution Dialog === *)
Ask_Human ≜ <<
  [field: "confirm",
   prompt: "Update WP {id}? Operation: {op}. Shows affected units and changes.",
   required: ⊤]
>>

(* === Post-Execution Routing === *)
(* G_ij bridges — formal handoff contracts *)
Routing ≜ [
  G(update-work-plan, proceed-work):
    units_pending > 0 → B_update.wp_path feeds A_proceed.wp_path,
  G(update-work-plan, verify-work):
    units_pending = 0 ∧ operation ≠ abort → all units terminal, feeds A_verify.wp_path,
  G(update-work-plan, archive-work):
    units_pending = 0 ∧ operation = abort → everything aborted, feeds A_archive.wp_path
]
',
  '["core-skill", "update-work-plan", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);

INSERT OR IGNORE INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
VALUES (
  'seed-verify-work',
  'gem2-studio',
  'prompt',
  'verify-work SKILL.md',
  '---
name: verify-work
description: >
  (Who) AI agent (e.g., Claude Code).
  (What) Verify results against CONTRACTs — determine STATE (SUCCESS|FAILURE) per unit.
  Structural verification (L0) + formal SOUND evaluation (L1 DEFAULT).
  (When) After each unit completes (inline from /proceed-work) or after all units complete (batch).
  (Where) WP file in .gem-squared/work-plan/ + verification log in .gem-squared/verify-work-logs/ (L0);
  gem2_sound via gem2-studio MCP for formal soundness audit (L1 DEFAULT).
  (Why) Structural drift detection between CONTRACT.B and Result.
  L1 adds boundary soundness analysis (SIMP, CONS, FALS, INCV, CTRST) that L0 cannot perform.
argument-hint: "[WP path or work title] [unit_index]"
metadata:
  author: David Seo of GEM².AI
  version: 0.5.2-L1
  layer: L1
  license: CC-BY-4.0
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash(date *)
  - Bash(mkdir *)
  - gem2_sound
  - gem2_task_update
---

(* TPMN SKILL — verify-work — L1 *)

(* === Layers === *)
L0 ≜ "Structural verification only: field coverage, type conformance, constraint satisfaction.
      Reads WP file, compares Result vs CONTRACT.B, writes State to WP file, writes log"
L1 ≜ "DEFAULT: structural verification + gem2_sound formal soundness audit + gem2_task_update sync.
      FALLBACK: degrade to L0 when gem2-studio MCP is unreachable"

(* === Grounding 5W === *)
Grounding_5W ≜ [
  who:   "AI agent — auto-proceed after CONTRACT comparison, no human input required",
  what:  "Verify unit results against CONTRACTs — structural check (L0) + SOUND audit (L1 DEFAULT).
          Determine STATE (SUCCESS|FAILURE) per unit. Write to WP file + verification log",
  when:  "After each unit completes (SINGLE mode, inline from /proceed-work) or
          after all units complete (BATCH mode, at end-of-WP or manually by human)",
  where: "L0: WP file in .gem-squared/work-plan/ (read CONTRACT + Result, write State).
          Verification log in .gem-squared/verify-work-logs/{wp_id}.md.
          L1 DEFAULT: additionally gem2_sound(content, session_context, domain) for formal soundness,
          gem2_task_update(task_id, state) for remote state sync",
  why:   "Structural drift detection between CONTRACT.B and Result — binary verdict per unit.
          L1 SOUND audit adds five-principle boundary soundness (SIMP, CONS, FALS, INCV, CTRST)
          that catches semantic violations invisible to structural checks alone"
]

(* === Modes === *)
(* verify-work operates in two modes, determined by unit_index input:        *)
(*   SINGLE: verify one unit (called inline by /proceed-work after each UC)  *)
(*   BATCH:  verify all units (called at end-of-WP or manually by human)     *)
MODE ≜ unit_index ≠ ⊥ ? SINGLE : BATCH

(* === Input === *)
A ≜ [
  project_slug: 𝕊,
  wp_path: Path?,                      (* direct path to WP file, or ⊥ *)
  work_title: 𝕊?,                     (* search term if wp_path not given *)
  unit_index: ℕ?,                     (* ⊥ = BATCH (all units). ≠ ⊥ = SINGLE (one unit) *)
  session_context: 𝕊?                 (* L1: core takeaways from session, for gem2_sound *)
]

(* === Output === *)
B ≜ [
  project_slug: 𝕊,
  wp_path: Path,
  wp_id: 𝕊,
  mode: {SINGLE, BATCH},
  layer: {DEFAULT, FALLBACK},          (* which execution path was taken *)
  overall_state: {SUCCESS, FAILURE},   (* SINGLE: state of that unit. BATCH: aggregate *)
  unit_results: Seq([
    unit_index: ℕ,
    unit_title: 𝕊,
    state: {SUCCESS, FAILURE},
    detail: 𝕊,                        (* what passed or what specifically failed *)
    sound_verdict: {PASS, FAIL}?,      (* L1 DEFAULT only: gem2_sound composite result *)
    sound_audits: 𝕊?                  (* L1 DEFAULT only: per-principle SIMP/CONS/FALS/INCV/CTRST *)
  ]),
  failures: Seq(𝕊)?,                  (* failed unit titles + reasons, ⊥ if all SUCCESS *)
  log_path: Path,                      (* .gem-squared/verify-work-logs/{wp_id}.md *)
  verified_at: 𝕊                      (* ISO8601 *)
]

(* === Precondition === *)
P ≜ project_slug ≠ ⊥
    ∧ (wp_path ≠ ⊥ ∨ work_title ≠ ⊥)
    ∧ (MODE = SINGLE
       ⟹ unit[unit_index].STATUS = COMPLETED)          (* only that unit must be done *)
    ∧ (MODE = BATCH
       ⟹ all unit STATUS ∈ {COMPLETED, ABORTED})       (* all units must be terminal *)

(* === Negative Contract === *)
¬B ≜ [
  ⊢ NEVER fix failures — report only,
  ⊢ NEVER re-execute work — that is /proceed-work mandate,
  ⊢ NEVER modify work output or Result fields — verify against what exists,
  ⊢ NEVER archive — that is /archive-work mandate,
  ⊢ NEVER apply subjective judgment — verify against CONTRACTs only,
  ⊢ NEVER use gem2_sound verdict to OVERRIDE structural check — SOUND enriches, does not replace,
  ⊢ NEVER block on gem2-studio MCP failure — degrade to FALLBACK silently
]

(* === Transform === *)
F ≜ <<
  1. Record verified_at timestamp. Determine mode.
     Identify WP:
       IF wp_path provided → read that WP.
       IF only work_title → search work-plan/ for matching WP.
     Verify precondition:
       SINGLE: unit[unit_index].STATUS = COMPLETED (that unit has a Result to verify).
         WP may be IN_PROGRESS — this is expected during inline verification.
       BATCH: all unit STATUS ∈ {COMPLETED, ABORTED} (all units terminal).
         IF not met → STOP, report: "WP not ready for batch verification."

  2. Select units to verify:
       SINGLE: verify only unit[unit_index].
       BATCH: verify all COMPLETED units. Skip ABORTED units (no result to verify).
     Extract each target unit''s CONTRACT (A, B, P) and Result.

  3. FOR each target unit, evaluate STATE:
     (* --- Structural verification (ALWAYS — both DEFAULT and FALLBACK) --- *)
       Let b = actual Result (recorded by /proceed-work).
       Let B = CONTRACT.B (expected output specification).
       Let P = CONTRACT.P (preconditions and constraints).

       (* Verification predicate *)
       unit.state = SUCCESS ⟺
         (∀ field ∈ B: b[field] ≠ ⊥ ∧ type(b[field]) = B[field].type)
         ∧ P(a, b) holds

       In practice:
         - Field coverage: ∀ field ∈ B → b[field] ≠ ⊥ (every contracted output field exists in result)
         - Type conformance: type(b[field]) = B[field].type (result types match contract types)
         - Constraint satisfaction: P(a, b) (preconditions and invariants hold against actual input/output)
       IF any predicate fails → unit.state = FAILURE. Record which predicate failed and why.

     (* --- SOUND evaluation (DEFAULT only — degrades to skip on FALLBACK) --- *)
       DEFAULT (gem2-studio MCP reachable):
         Combine Result text with CONTRACT.B text into a single evaluation payload.
         Call gem2_sound(
           content   = "{CONTRACT.B}\n---\n{Result}",
           session_context = session_context ∨ "Verifying unit {unit_index}: {unit_title}",
           domain    = "sdlc"
         ).
         Receive SoundReport: composite (PASS|FAIL), per-principle audits
           (SIMP, CONS, FALS, INCV, CTRST — each with verdict + reasoning).
         Set unit.sound_verdict = SoundReport.composite.
         Set unit.sound_audits = formatted per-principle summary.
         Merge SOUND audit with structural verification:
           IF structural = SUCCESS ∧ sound_verdict = FAIL →
             unit.state remains SUCCESS (SOUND enriches, does not override).
             Record SOUND failures as advisory warnings in detail.
           IF structural = FAILURE → unit.state = FAILURE regardless of SOUND.
           SOUND audit is informational — structural check is authoritative.
       FALLBACK (gem2-studio MCP unreachable):
         unit.sound_verdict = ⊥.
         unit.sound_audits = ⊥.
         Set layer = FALLBACK.

  4. Determine overall STATE:
       SINGLE: overall_state = unit[unit_index].state.
       BATCH: overall_state = SUCCESS iff all verified units SUCCESS, else FAILURE.
     Edit WP file:
       Write `- State: SUCCESS` or `- State: FAILURE` per verified unit-work.
       Do NOT write WP-level STATE in header — that is /archive-work mandate.
     DEFAULT (gem2-studio MCP reachable):
       Call gem2_task_update(task_id, state=overall_state).
     FALLBACK: skip — WP file is source of truth.

  5. Write verification log:
       mkdir -p .gem-squared/verify-work-logs/
       SINGLE: append section to .gem-squared/verify-work-logs/{wp_id}.md:
         IF file does not exist → create with WP header, then append unit section.
         IF file exists → append unit section at end (before Summary if present).

         ## Unit {N}: {title} — {STATE} (verified {verified_at})
         ### CONTRACT.B (expected)
         {full CONTRACT.B text}
         ### Result (actual)
         {full Result text}
         ### Structural Judgment
         {detailed comparison — what matched, what didn''t, why STATE was determined}
         ### SOUND Audit (L1 DEFAULT only — omit section if FALLBACK)
         **Composite:** {PASS|FAIL}
         | Principle | Verdict | Detail |
         |-----------|---------|--------|
         | SIMP      | ...     | ...    |
         | CONS      | ...     | ...    |
         | FALS      | ...     | ...    |
         | INCV      | ...     | ...    |
         | CTRST     | ...     | ...    |

       BATCH: overwrite .gem-squared/verify-work-logs/{wp_id}.md with full log:
         # Verification Log: {wp_id}
         **WP:** {wp_title} | **Verified:** {verified_at} | **Layer:** {DEFAULT|FALLBACK}
         **Overall:** {overall_state} | **Units verified:** {count} | **Skipped (ABORTED):** {count}

         ## Unit 1: {title} — {STATE}
         ### CONTRACT.B (expected)
         {full CONTRACT.B text}
         ### Result (actual)
         {full Result text}
         ### Structural Judgment
         {detailed comparison}
         ### SOUND Audit (if DEFAULT)
         {per-principle table}

         (repeat for each verified unit)

         ## Summary
         {overall assessment}
         {IF DEFAULT: aggregate SOUND observations across units}

     Output B.
>>

(* === Constraint === *)
CONSTRAINT ≜ [
  ⊢ NEVER fix failures — report only,
  ⊢ NEVER re-execute work — that is /proceed-work mandate,
  ⊢ NEVER modify work output or Result fields — verify against what exists,
  ⊢ NEVER archive — that is /archive-work mandate,
  ⊢ NEVER apply subjective judgment — verify against CONTRACTs only,
  ⊢ Verification is binary per unit: SUCCESS or FAILURE — no partial credit,
  ⊢ SINGLE mode verifies ONE unit — does not touch other units'' State fields,
  ⊢ SOUND audit enriches judgment — does NOT override structural verification,
  ⊢ gem2-studio MCP sync failure does NOT block verification — WP file is the source of truth
]

(* === Invariant === *)
INV ≜ [
  ⊢ STATE is written to WP file alongside STATUS — separate concerns (STATUS = did it run, STATE = did it pass),
  ⊢ BATCH: overall_state = FAILURE if ANY unit fails — no partial success,
  ⊢ SINGLE: overall_state reflects only the target unit,
  ⊢ Failures include specific reasons — not just pass/fail,
  ⊢ ABORTED units are skipped (no result to verify) — they do not affect overall STATE,
  ⊢ WP file stores the verdict (State field) — routing signal,
  ⊢ Log file stores the full reasoning (CONTRACT.B vs Result comparison + SOUND audit) — decision support,
  ⊢ One log file per WP — SINGLE appends, BATCH overwrites,
  ⊢ WP file is the source of truth for STATE,
  ⊢ B.layer reflects actual execution path taken (DEFAULT or FALLBACK),
  ⊢ DEFAULT: structural verification + gem2_sound audit + gem2_task_update sync,
  ⊢ FALLBACK: structural verification only — fully functional without gem2-studio MCP,
  ⊢ MANDATE BOUNDARY: verification and reporting — NOT work execution, NOT archiving, NOT fixing failures
]

(* === Post-Execution Routing === *)
Routing ≜ [
  MODE = SINGLE → G₁₁: return to caller (/proceed-work handles next step based on state),
  MODE = BATCH ∧ overall_state = SUCCESS  → G₁₂: /archive-work (persist the win),
  MODE = BATCH ∧ overall_state = FAILURE  → G₁₃: present failures to human:
    → human may /proceed-work to retry specific unit,
    → human may /archive-work with FAILURE state accepted
]
',
  '["core-skill", "verify-work", "L1"]',
  'gem2-studio-seed',
  1,
  '{}'
);
