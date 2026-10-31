import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import fs from "fs";
import path from "path";
import { exec } from "child_process";
import dotenv from "dotenv";
import mysql from "mysql2/promise";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const PROJECT_ROOT = path.resolve(__dirname, "..");

// Load .env dari root Cinembrot (quiet=true agar tidak ada noise di stdio MCP)
dotenv.config({ path: path.join(PROJECT_ROOT, ".env"), quiet: true });

const server = new Server(
  {
    name: "cinembrot-diagnostics",
    version: "1.1.0",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Read-only MySQL Pool (MariaDB Cinembrot di host server taoen45 via WireGuard 55.0.0.157 atau localhost)
let dbPool = null;
function getDbPool() {
  if (!dbPool) {
    dbPool = mysql.createPool({
      host: process.env.DB_HOST || "55.0.0.157",
      port: parseInt(process.env.DB_PORT || "3306", 10),
      user: process.env.DB_USER || "root",
      password: process.env.DB_PASS || "45root45",
      database: process.env.DB_NAME || "cinembrot",
      waitForConnections: true,
      connectionLimit: 3,
      queueLimit: 0,
      connectTimeout: 8000,
    });
  }
  return dbPool;
}

function runExec(cmd, cwd = PROJECT_ROOT, maxBuffer = 1024 * 1024 * 4, timeout = 30000) {
  return new Promise((resolve) => {
    exec(cmd, { cwd, maxBuffer, timeout }, (error, stdout, stderr) => {
      resolve({
        stdout: stdout ? stdout.trim() : "",
        stderr: stderr ? stderr.trim() : "",
        exitCode: error ? error.code || 1 : 0,
      });
    });
  });
}

// Informasi Lengkap GitHub & Infrastruktur Server taoen45
const CINEMBROT_GITHUB_INFO = {
  project: "CINEMBROT — Movie Engine, Streamer & Torrent Subtitle Downloader",
  remotes: {
    fork: "https://github.com/taoen45/cinembrot.git",
    origin: "https://github.com/supermabur/cinembrot.git",
  },
  activeBranch: "main",
  localWorkspace: "D:/www/Cinembrot",
  productionServer: {
    sshHost: "taoen45 (WireGuard 55.0.0.157, LAN 192.168.5.115)",
    repoPath: "/home/taoen45/docker/html",
    dockerService: "app (Golang binary running on :8080)",
    caddyProxy: "Caddy (:80 & :81 -> Go :8080 & :8081)",
    cloudflareDomains: {
      mainStreaming: "cinembrot.web.id (proxy :80 -> :8080)",
      infoLanding: "cinembrot.my.id (proxy :81 -> :8081)",
    },
    cronSchedule: {
      dailyScrape: "05:00 WIB & 17:00 WIB (/home/taoen45/docker/host/cinembrot-daily-scrape.sh)",
      nightlyUpgrade: "03:40 WIB (homelab-nightly-upgrade.timer)",
      nightlyReboot: "04:00 WIB (homelab-nightly-reboot.timer)",
    },
    timezone: "Asia/Jakarta (WIB, UTC+7)",
  },
  protectedFilesNeverCommit: [
    ".env",
    ".env.*",
    "cinembrot.exe",
    "mcp-server/node_modules/",
  ],
};

// Pengetahuan Penting Proyek & Guardrails (AGENTS.md + INSTRUKSI-SERVER.md)
const CINEMBROT_KNOWLEDGE = {
  projectName: "CINEMBROT",
  architecture3Layers: {
    layer1_context: "AI Memory (.ai-memory/) + RAG Search (search_project_rag) + Modular Rules (.cursor/rules/)",
    layer2_execution: "MCP Tools (cinembrot-diagnostics) + Terminal terkontrol (Go build/run, git)",
    layer3_validation: "Playwright E2E (tests/e2e/smoke.spec.js) + Go Unit Tests via tool 'run_tests' (compact output)",
  },
  architecture: "Golang 1.22+ Native (No PHP required) + GORM + MariaDB + Redis + Caddy + Cloudflare Tunnel",
  github: CINEMBROT_GITHUB_INFO,
  guardrails: [
    "DILARANG KERAS memodifikasi atau menghapus data database tanpa izin eksplisit user (DATABASE WAJIB IZIN).",
    "Gunakan Bahasa Indonesia dalam semua percakapan, instruksi, dan dokumentasi.",
    "Perubahan sistem/fitur/port WAJIB dicatat di INSTRUKSI-SERVER.md pada bagian Riwayat Perubahan.",
    "Jangan membocorkan secret .env, token Cloudflare, atau kunci WireGuard ke chat/commit.",
    "Server Golang independen, TIDAK membutuhkan runtime PHP.",
    "Semua template HTML mendukung live auto-reload saat disimpan.",
  ],
  logSources: {
    dockerGoApp: "docker logs cinembrot-app-1 (atau docker logs docker-app-1)",
    caddyReverseProxy: "docker logs cinembrot-caddy-1 (atau journalctl -u caddy)",
    cronScraper: "/var/log/cinembrot/daily-scrape.log atau crontab output",
    systemMaintenance: "/var/log/homelab/nightly-upgrade.log & /var/log/homelab/nightly-reboot.log",
    cloudflared: "journalctl -u cloudflared -n 50 --no-pager",
  },
  coreFeatures: [
    "Multi-Server Streaming Embed (VidSrc, AutoEmbed, 2Embed, VidLink, SuperEmbed, MultiEmbed, NontonGo)",
    "Dwibahasa ID & EN dengan auto-translate sinopsis",
    "Automated Scraper (TMDb, Jikan Anime, TMDb TV Asian Drama, Internet Archive, YTS)",
    "Dual Domain: cinembrot.web.id (Streaming Hub) & cinembrot.my.id (Landing Info)",
    "Local WebP Converter untuk Poster & Backdrop (Hemat bandwidth & storage)",
    "Keamanan HTTP Security Headers, In-Memory Rate Limiting, & Regex Sanitization",
  ],
};

server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: "get_project_knowledge",
        description:
          "Mengambil ringkasan arsitektur 3-layer, stack Golang, aturan wajib AGENTS.md, jadwal cron, konfigurasi Cloudflare dual-domain, dan guardrails keamanan Cinembrot.",
        inputSchema: {
          type: "object",
          properties: {},
        },
      },
      {
        name: "search_project_rag",
        description:
          "Layer 1 Token-Saving RAG: Mencari potongan pengetahuan proyek dari .ai-memory/ dan file dokumentasi penting secara terindeks untuk menghemat token konteks AI Agent (tanpa dump file utuh).",
        inputSchema: {
          type: "object",
          properties: {
            query: {
              type: "string",
              description: "Kata kunci pencarian (misal: 'database safety', 'stream server', 'ad cooldown', 'webp').",
            },
            source: {
              type: "string",
              enum: ["all", "ai-memory", "docs"],
              description: "Sumber pencarian (default: 'all').",
            },
            maxChunks: {
              type: "number",
              description: "Maksimal potongan hasil (default: 5, max: 10).",
            },
          },
          required: ["query"],
        },
      },
      {
        name: "run_tests",
        description:
          "Layer 3 Compact Validation Runner: Menjalankan Go test ('go') atau Playwright E2E ('playwright') dan mengembalikan ringkasan status (passed/failed/failures) secara hemat token tanpa memenuhi layar terminal.",
        inputSchema: {
          type: "object",
          properties: {
            suite: {
              type: "string",
              enum: ["go", "playwright"],
              description: "Framework testing yang dijalankan: 'go' (Go unit tests) atau 'playwright' (E2E browser tests).",
            },
            filter: {
              type: "string",
              description: "Filter nama package/test (opsional, misal './server' atau 'smoke.spec.js').",
            },
          },
          required: ["suite"],
        },
      },
      {
        name: "diagnose_server_logs",
        description:
          "Layer 2 Tool: Mendiagnosis log server langsung dari host taoen45 atau container Docker Cinembrot (Go App, Caddy, Scraper, Nightly Maintenance, Cloudflared) via SSH/remote command tanpa menghabiskan context token.",
        inputSchema: {
          type: "object",
          properties: {
            target: {
              type: "string",
              description:
                "Sumber log yang ingin diperiksa: 'app' (Docker Go cinembrot), 'caddy' (Reverse proxy), 'scraper' (Cron scraper 05:00 & 17:00), 'maintenance' (Upgrade/Reboot subuh), 'cloudflared' (Tunnel status), atau 'custom'.",
              enum: ["app", "caddy", "scraper", "maintenance", "cloudflared", "custom"],
            },
            lines: {
              type: "number",
              description: "Jumlah baris log terakhir yang ingin ditampilkan (default: 50, maks: 200).",
            },
            filterKeyword: {
              type: "string",
              description: "Filter kata kunci pencarian (misal: 'error', 'fatal', 'panic', 'scrape', 'timeout').",
            },
            customCommand: {
              type: "string",
              description: "Perintah kustom jika target='custom' (khusus diagnosa read-only log).",
            },
          },
          required: ["target"],
        },
      },
      {
        name: "inspect_db_schema_safe",
        description:
          "Layer 2 Tool: Melakukan inspeksi skema MariaDB Cinembrot secara aman (Read-Only) seperti daftar tabel, struktur kolom, status migration, atau ringkasan total data tanpa query write.",
        inputSchema: {
          type: "object",
          properties: {
            mode: {
              type: "string",
              description:
                "Mode inspeksi: 'tables' (daftar seluruh tabel & jumlah baris), 'describe' (struktur kolom tabel tertentu), 'stats' (statistik movie, anime, drama, download link), 'recent_movies' (10 film terbaru).",
              enum: ["tables", "describe", "stats", "recent_movies"],
            },
            tableName: {
              type: "string",
              description: "Nama tabel yang ingin di-describe (contoh: 'movies', 'settings', 'download_links', 'actors').",
            },
          },
          required: ["mode"],
        },
      },
      {
        name: "git_project_status",
        description:
          "Mengecek status Git repositori Cinembrot lokal & fork/origin GitHub (branch aktif, commit terbaru, file modified, remote tracking, perbandingan status remote).",
        inputSchema: {
          type: "object",
          properties: {
            logCount: {
              type: "number",
              description: "Jumlah commit terakhir yang ingin dilihat (default: 8, max: 25).",
            },
          },
        },
      },
    ],
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args = {} } = request.params;

  try {
    switch (name) {
      case "get_project_knowledge": {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(CINEMBROT_KNOWLEDGE, null, 2),
            },
          ],
        };
      }

      case "search_project_rag": {
        const query = String(args.query || "").toLowerCase();
        const source = args.source || "all";
        const maxChunks = Math.min(Math.max(parseInt(args.maxChunks || 5, 10), 1), 10);

        const targetDirs = [];
        if (source === "all" || source === "ai-memory") {
          targetDirs.push(path.join(PROJECT_ROOT, ".ai-memory"));
        }
        if (source === "all" || source === "docs") {
          targetDirs.push(path.join(PROJECT_ROOT, ".cursor", "rules"));
        }

        const keywords = query.split(/\s+/).filter((w) => w.length > 1);
        const results = [];

        function scanDir(dir) {
          if (!fs.existsSync(dir)) return;
          const items = fs.readdirSync(dir, { withFileTypes: true });
          for (const item of items) {
            const fullPath = path.join(dir, item.name);
            if (item.isDirectory() && !item.name.startsWith(".")) {
              scanDir(fullPath);
            } else if (item.isFile() && (item.name.endsWith(".md") || item.name.endsWith(".mdc"))) {
              const relPath = path.relative(PROJECT_ROOT, fullPath).replace(/\\/g, "/");
              try {
                const content = fs.readFileSync(fullPath, "utf-8");
                const sections = content.split(/\n(?=##?\s+)/);
                for (const section of sections) {
                  const secLower = section.toLowerCase();
                  let score = 0;
                  for (const kw of keywords) {
                    if (secLower.includes(kw)) score += 1;
                  }
                  if (score > 0) {
                    const lines = section.trim().split("\n");
                    const title = lines[0] ? lines[0].replace(/^#+\s*/, "") : item.name;
                    const preview = lines.slice(0, 15).join("\n");
                    results.push({
                      file: relPath,
                      sectionTitle: title,
                      score,
                      chunkText: preview,
                    });
                  }
                }
              } catch {
                // Ignore read errors
              }
            }
          }
        }

        for (const dir of targetDirs) {
          scanDir(dir);
        }

        // Cari juga pada README.md dan INSTRUKSI-SERVER.md secara hemat potongan
        if (source === "all") {
          for (const extraFile of ["README.md", "INSTRUKSI-SERVER.md"]) {
            const fullPath = path.join(PROJECT_ROOT, extraFile);
            if (fs.existsSync(fullPath)) {
              try {
                const content = fs.readFileSync(fullPath, "utf-8");
                const sections = content.split(/\n(?=##?\s+)/);
                for (const section of sections) {
                  const secLower = section.toLowerCase();
                  let score = 0;
                  for (const kw of keywords) {
                    if (secLower.includes(kw)) score += 1;
                  }
                  if (score > 0) {
                    const lines = section.trim().split("\n");
                    const title = lines[0] ? lines[0].replace(/^#+\s*/, "") : extraFile;
                    const preview = lines.slice(0, 15).join("\n");
                    results.push({
                      file: extraFile,
                      sectionTitle: title,
                      score,
                      chunkText: preview,
                    });
                  }
                }
              } catch {
                // ignore
              }
            }
          }
        }

        results.sort((a, b) => b.score - a.score);
        const topResults = results.slice(0, maxChunks);

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(
                {
                  query,
                  found: topResults.length,
                  results: topResults,
                },
                null,
                2
              ),
            },
          ],
        };
      }

      case "run_tests": {
        const suite = args.suite;
        const filter = args.filter ? String(args.filter).trim() : null;

        if (suite === "go") {
          const filterArg = filter ? ` ${filter}` : " ./server";
          const cmd = `go test${filterArg} -v`;
          const res = await runExec(cmd, PROJECT_ROOT, 1024 * 1024 * 2, 60000);
          const output = (res.stdout + "\n" + res.stderr).trim();
          const passed = res.exitCode === 0 && !output.includes("FAIL") && !output.includes("[build failed]");

          const lines = output.split(/\r?\n/);
          const passLines = lines.filter((l) => l.startsWith("--- PASS:") || l.startsWith("ok  \t"));
          const failLines = lines.filter((l) => l.startsWith("--- FAIL:") || l.includes("FAIL\t") || l.includes("undefined:"));

          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(
                  {
                    suite: "go",
                    filter: filter || "./server",
                    passed,
                    summary: {
                      passedCount: passLines.length,
                      failedCount: failLines.length,
                      passedTests: passLines.slice(-5),
                      failurePreview: failLines.slice(0, 5),
                    },
                    rawExcerpt: passed ? "All selected tests passed." : failLines.slice(0, 8).join("\n"),
                  },
                  null,
                  2
                ),
              },
            ],
          };
        }

        if (suite === "playwright") {
          const filterArg = filter ? ` tests/e2e/${filter}` : " tests/e2e/smoke.spec.js";
          const cmd = `npx --yes playwright test${filterArg}`;
          const res = await runExec(cmd, PROJECT_ROOT, 1024 * 1024 * 2, 60000);
          const output = (res.stdout + "\n" + res.stderr).trim();
          const passed = res.exitCode === 0;

          const lines = output.split(/\r?\n/);
          const summaryLine = lines.reverse().find((l) => /\d+\s+(passed|failed|skipped)/i.test(l)) || "";
          const failures = lines.filter((l) => /Error:|failed/i.test(l)).slice(0, 5);

          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(
                  {
                    suite: "playwright",
                    passed,
                    summary: summaryLine.trim() || (passed ? "Tests passed" : "Tests failed/skipped"),
                    failurePreview: failures,
                  },
                  null,
                  2
                ),
              },
            ],
          };
        }

        return { content: [{ type: "text", text: `Suite test tidak dikenal: ${suite}` }] };
      }

      case "diagnose_server_logs": {
        const target = args.target || "app";
        const lines = Math.min(Math.max(10, args.lines || 50), 200);
        const filter = (args.filterKeyword || "").trim();

        let remoteCmd = "";
        switch (target) {
          case "app":
            remoteCmd = `docker logs --tail ${lines} \$(docker ps -q -f name=app -f name=cinembrot | head -n 1) 2>&1`;
            break;
          case "caddy":
            remoteCmd = `docker logs --tail ${lines} \$(docker ps -q -f name=caddy | head -n 1) 2>&1`;
            break;
          case "scraper":
            remoteCmd = `tail -n ${lines} /var/log/cinembrot/daily-scrape.log 2>/dev/null || cat /home/taoen45/docker/host/cinembrot-daily-scrape.sh 2>&1`;
            break;
          case "maintenance":
            remoteCmd = `tail -n ${lines} /var/log/homelab/nightly-upgrade.log 2>/dev/null || journalctl -u homelab-nightly-upgrade.service -n ${lines} --no-pager`;
            break;
          case "cloudflared":
            remoteCmd = `journalctl -u cloudflared -n ${lines} --no-pager`;
            break;
          case "custom":
            if (!args.customCommand) {
              return { isError: true, content: [{ type: "text", text: "customCommand wajib diisi jika target='custom'" }] };
            }
            remoteCmd = args.customCommand;
            break;
        }

        // Jalankan via SSH ke taoen45 jika host terjangkau
        const sshHost = "taoen45";
        const executeCmd = `ssh -o BatchMode=yes -o ConnectTimeout=6 ${sshHost} "${remoteCmd}"`;
        const res = await runExec(executeCmd);

        let outputText = res.stdout || res.stderr;
        if (!outputText && res.exitCode !== 0) {
          const fallbackCmd = `ssh -o BatchMode=yes -o ConnectTimeout=6 taoen45@55.0.0.157 "${remoteCmd}"`;
          const fbRes = await runExec(fallbackCmd);
          outputText = fbRes.stdout || fbRes.stderr;
        }

        if (filter && outputText) {
          const regex = new RegExp(filter, "i");
          outputText = outputText
            .split(/\r?\n/)
            .filter((line) => regex.test(line))
            .join("\n");
        }

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(
                {
                  target,
                  linesRequested: lines,
                  filterApplied: filter || null,
                  executedCommand: remoteCmd,
                  outputSample: outputText ? outputText.slice(-10000) : "Tidak ada output atau koneksi SSH timeout.",
                },
                null,
                2
              ),
            },
          ],
        };
      }

      case "inspect_db_schema_safe": {
        const mode = args.mode || "tables";
        const pool = getDbPool();

        if (mode === "tables") {
          const [tables] = await pool.query(
            `SELECT TABLE_NAME, TABLE_ROWS, DATA_LENGTH, INDEX_LENGTH, UPDATE_TIME 
             FROM information_schema.TABLES 
             WHERE TABLE_SCHEMA = DATABASE() 
             ORDER BY TABLE_ROWS DESC`
          );
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify({ database: process.env.DB_NAME || "cinembrot", tables }, null, 2),
              },
            ],
          };
        }

        if (mode === "describe") {
          const tableName = (args.tableName || "").replace(/[^a-zA-Z0-9_]/g, "");
          if (!tableName) {
            return { isError: true, content: [{ type: "text", text: "tableName wajib diisi untuk mode describe" }] };
          }
          const [columns] = await pool.query(`DESCRIBE \`${tableName}\``);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify({ table: tableName, columns }, null, 2),
              },
            ],
          };
        }

        if (mode === "stats") {
          const [[movieCount]] = await pool.query("SELECT COUNT(*) as total_movies FROM movies");
          const [[animeCount]] = await pool.query("SELECT COUNT(*) as total_anime FROM movies WHERE is_anime = 1");
          const [[dramaCount]] = await pool.query("SELECT COUNT(*) as total_drama FROM movies WHERE is_drama = 1");
          const [[linkCount]] = await pool.query("SELECT COUNT(*) as total_downloads FROM download_links");
          const [[streamCount]] = await pool.query("SELECT COUNT(*) as total_streams FROM stream_servers");

          return {
            content: [
              {
                type: "text",
                text: JSON.stringify(
                  {
                    total_movies: movieCount.total_movies,
                    total_anime: animeCount.total_anime,
                    total_drama: dramaCount.total_drama,
                    total_download_links: linkCount.total_downloads,
                    total_stream_servers: streamCount.total_streams,
                  },
                  null,
                  2
                ),
              },
            ],
          };
        }

        if (mode === "recent_movies") {
          const [rows] = await pool.query(
            "SELECT id, title, original_title, year, rating, quality, is_anime, is_drama, created_at FROM movies ORDER BY id DESC LIMIT 10"
          );
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify({ recentMovies: rows }, null, 2),
              },
            ],
          };
        }

        return { content: [{ type: "text", text: `Mode tidak dikenal: ${mode}` }] };
      }

      case "git_project_status": {
        const logCount = Math.min(Math.max(1, args?.logCount || 8), 25);

        const [branchRes, statusRes, diffStatRes, logRes, remoteRes, headRes] = await Promise.all([
          runExec("git branch -vv --no-color"),
          runExec("git status -s"),
          runExec("git diff --stat"),
          runExec(`git log -n ${logCount} --oneline --decorate`),
          runExec("git remote -v"),
          runExec("git rev-parse --short HEAD"),
        ]);

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(
                {
                  projectGithub: CINEMBROT_GITHUB_INFO,
                  localGit: {
                    headCommit: headRes.stdout,
                    branches: branchRes.stdout.split(/\r?\n/).filter(Boolean),
                    remotes: remoteRes.stdout.split(/\r?\n/).filter(Boolean),
                    modifiedFiles: statusRes.stdout ? statusRes.stdout.split(/\r?\n/).filter(Boolean) : [],
                    diffStat: diffStatRes.stdout ? diffStatRes.stdout.split(/\r?\n/).filter(Boolean) : [],
                    recentCommits: logRes.stdout.split(/\r?\n/).filter(Boolean),
                  },
                },
                null,
                2
              ),
            },
          ],
        };
      }

      default:
        throw new Error(`Tool tidak dikenal: ${name}`);
    }
  } catch (error) {
    return {
      isError: true,
      content: [{ type: "text", text: `Error pada ${name}: ${error.message}` }],
    };
  }
});

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((err) => {
  console.error("Fatal error starting Cinembrot MCP server:", err);
  process.exit(1);
});
