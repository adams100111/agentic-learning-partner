/**
 * PyLearn side of ALP's closed-loop smoke (internal/e2e/closedloop_test.go,
 * ticket #75, spec #66 Seam 3). Run by the smoke from a PyLearn checkout's
 * apps/web directory with PyLearn's own tsx, so every write goes through
 * PyLearn's real code:
 *
 *   tsx <this file> bootstrap --target <id> --title <title> --pack <domain>@<range> --mapping-out <file>
 *     The curriculum export and empty mapping of a target before any content
 *     exists, built by the same library functions `alp:bootstrap` uses
 *     (newTargetMapping, newTargetCurriculum). Writes the mapping to
 *     --mapping-out (never into the repo) and prints the export.
 *
 *   tsx <this file> seed-quiz --user <id> --lesson <id> --concept <concept> --answers <json> --now-ms <ms>
 *     Seeds synthetic learner activity into the database at DATABASE_URL: the
 *     user through roster.createUser (once) and the quiz answers through
 *     persistQuizAnswers, exactly as POST /api/quiz persists them. The quiz id
 *     is the quiz's concept, as the timeline compiler persists it.
 *
 * PyLearn modules are loaded from the current directory (apps/web), so their
 * `@/` imports resolve through apps/web/tsconfig.json.
 */
import { writeFileSync } from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const web = process.cwd();
const load = (rel: string) => import(pathToFileURL(path.join(web, rel)).href);

function arg(argv: readonly string[], flag: string): string {
  const i = argv.indexOf(flag);
  const value = i === -1 ? undefined : argv[i + 1];
  if (value === undefined) throw new Error(`${flag} is required`);
  return value;
}

async function bootstrap(argv: readonly string[]): Promise<void> {
  const { newTargetCurriculum, newTargetMapping } = await load("lib/courses/alp-authoring.ts");
  const target = arg(argv, "--target");
  const pack = arg(argv, "--pack");
  const at = pack.indexOf("@");
  const mappingText: string = newTargetMapping(target, [
    { domain: pack.slice(0, at), packVersion: pack.slice(at + 1) },
  ]);
  writeFileSync(arg(argv, "--mapping-out"), mappingText);
  const curriculum = newTargetCurriculum({
    target,
    title: arg(argv, "--title"),
    mappingRef: `content/alp/${target}.mapping.yaml`,
    mappingText,
  });
  process.stdout.write(`${JSON.stringify(curriculum, null, 2)}\n`);
}

async function seedQuiz(argv: readonly string[]): Promise<void> {
  const { getDb } = await load("db/index.ts");
  const { createUser, getProfile } = await load("lib/users/roster.ts");
  const { persistQuizAnswers } = await load("lib/reel/quiz-persist.ts");
  const db = getDb();
  const user = arg(argv, "--user");
  if (!(await getProfile(user, db))) {
    await createUser({ id: user, name: "Synthetic Smoke Learner", color: "#4f7cac" }, db);
  }
  const concept = arg(argv, "--concept");
  await persistQuizAnswers(db, user, {
    lessonId: arg(argv, "--lesson"),
    quizId: concept,
    concept,
    answers: JSON.parse(arg(argv, "--answers")),
    nowMs: Number(arg(argv, "--now-ms")),
  });
  process.stdout.write(`${JSON.stringify({ seeded: user })}\n`);
}

const [command, ...rest] = process.argv.slice(2);
const run = command === "bootstrap" ? bootstrap : command === "seed-quiz" ? seedQuiz : undefined;
if (!run) {
  console.error(`unknown command "${command ?? ""}" (bootstrap | seed-quiz)`);
  process.exit(2);
}
run(rest).catch((err) => {
  console.error(err instanceof Error ? `${err.name}: ${err.message}` : err);
  process.exit(1);
});
