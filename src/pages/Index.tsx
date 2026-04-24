import { useState } from "react";
import { Link } from "react-router-dom";
import { FolderGit2, GitBranch, RefreshCw, Eye } from "lucide-react";
import DocsLayout from "@/components/docs/DocsLayout";
import FeatureCard from "@/components/docs/FeatureCard";
import InstallBlock from "@/components/docs/InstallBlock";
import { VERSION } from "@/constants/index";

type Mode = "install" | "uninstall";

const COMMAND_TABS: Record<Mode, { label: string; command: string }[]> = {
  install: [
    {
      label: "Windows",
      command:
        "irm https://raw.githubusercontent.com/alimtvnetwork/gitmap-v7/main/install-quick.ps1 | iex",
    },
    {
      label: "Linux / macOS",
      command:
        "curl -fsSL https://raw.githubusercontent.com/alimtvnetwork/gitmap-v7/main/install-quick.sh | bash",
    },
  ],
  uninstall: [
    {
      label: "Windows",
      command:
        "irm https://raw.githubusercontent.com/alimtvnetwork/gitmap-v7/main/uninstall-quick.ps1 | iex",
    },
    {
      label: "Linux / macOS",
      command:
        "curl -fsSL https://raw.githubusercontent.com/alimtvnetwork/gitmap-v7/main/uninstall-quick.sh | bash",
    },
  ],
};

const MODES: { key: Mode; label: string; tooltip: string }[] = [
  {
    key: "install",
    label: "Install",
    tooltip: "Runs install-quick.ps1 / install-quick.sh — the one-line installer",
  },
  {
    key: "uninstall",
    label: "Uninstall",
    tooltip:
      "Runs uninstall-quick.ps1 / uninstall-quick.sh — the one-line uninstaller",
  },
];

const HomePage = () => {
  const [mode, setMode] = useState<Mode>("install");

  return (
    <DocsLayout>
      <section className="py-14 text-center">
        <div className="reveal">
          <div className="flex items-center justify-center gap-3 mb-4">
            <h1 className="text-4xl md:text-6xl font-heading font-bold docs-h1 text-shimmer tracking-tight">
              gitmap
            </h1>
            <span className="rounded-sm border border-border bg-card px-2 py-0.5 text-xs font-mono text-muted-foreground shadow-sm">
              {VERSION}
            </span>
          </div>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto mb-8 leading-relaxed font-sans">
            Scan a folder tree for Git repos, generate structured clone files, and
            re-clone the exact layout on any machine. Track, group, release, and
            manage repositories from a single CLI.
          </p>

          <div className="mx-auto mb-8 max-w-3xl rounded-xl bg-card/40 px-8 py-7 text-center backdrop-blur-sm">
            <div className="mb-6 flex items-center justify-center gap-2 pb-2">
              <span className="h-2.5 w-2.5 rounded-full bg-destructive/80" />
              <span className="h-2.5 w-2.5 rounded-full bg-primary/80" />
              <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/50" />
              <p className="ml-2 text-xs font-sans uppercase tracking-[0.18em] text-muted-foreground">
                Terminal quick actions
              </p>
            </div>

            {/* Mode tabs (Install / Uninstall) — primary axis */}
            <div
              role="tablist"
              aria-label="Install or uninstall"
              className="mx-auto mb-5 inline-flex gap-1 rounded-md border border-border bg-secondary/70 p-1"
            >
              {MODES.map((m) => {
                const active = m.key === mode;
                return (
                  <button
                    key={m.key}
                    role="tab"
                    aria-selected={active}
                    title={m.tooltip}
                    onClick={() => setMode(m.key)}
                    className={`btn-slide ${
                      active ? "" : "btn-slide-ghost"
                    } px-5 py-1.5 rounded-md text-sm font-heading font-semibold tracking-wide uppercase transition-all duration-300 ${
                      active
                        ? "bg-primary text-primary-foreground shadow-sm"
                        : "bg-transparent text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    {m.label}
                  </button>
                );
              })}
            </div>

            <div className="mx-auto max-w-2xl">
              <InstallBlock tabs={COMMAND_TABS[mode]} />
            </div>

            <p className="mx-auto mt-6 max-w-2xl text-xs text-muted-foreground font-sans leading-relaxed">
              Uninstall removes the <code className="font-mono text-foreground">gitmap</code> binary and its PATH entries, then prompts before deleting your data folder
              (<code className="font-mono text-foreground">%APPDATA%\gitmap</code> on Windows, <code className="font-mono text-foreground">~/.config/gitmap</code> on Linux/macOS).
              Pass <code className="font-mono text-foreground">--keep-data</code> to always keep it, or <code className="font-mono text-foreground">-y</code>/<code className="font-mono text-foreground">--yes</code> to skip the prompt.
            </p>
          </div>

          <div className="flex gap-4 justify-center">
            <Link
              to="/getting-started"
              className="btn-slide group relative rounded-sm border border-primary bg-primary px-6 py-2.5 font-heading text-sm font-medium text-primary-foreground shadow-sm hover:brightness-110 active:translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            >
              Get Started
            </Link>
            <Link
              to="/commands"
              className="btn-slide btn-slide-ghost group relative rounded-sm border border-border bg-card px-6 py-2.5 font-heading text-sm font-medium text-foreground hover:border-primary/40 hover:bg-secondary active:translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            >
              View Commands
            </Link>
          </div>
        </div>
      </section>

      <hr className="docs-hr" />

      <section className="reveal grid md:grid-cols-2 gap-4 py-8">
        <FeatureCard
          icon={FolderGit2}
          title="Scan & Map"
          description="Recursively discover Git repos, extract metadata, and output CSV/JSON/terminal views with clone scripts."
        />
        <FeatureCard
          icon={GitBranch}
          title="Clone & Restore"
          description="Re-clone the exact folder structure on a new machine from JSON, CSV, or text files with safe-pull and progress tracking."
        />
        <FeatureCard
          icon={RefreshCw}
          title="Release & Version"
          description="Create releases with tags, branches, changelogs, and semantic versioning — all from the command line."
        />
        <FeatureCard
          icon={Eye}
          title="Watch & Monitor"
          description="Live-refresh dashboard showing dirty/clean status, ahead/behind counts, and stash entries across all tracked repos."
        />
      </section>

      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "gitmap",
            applicationCategory: "DeveloperApplication",
            operatingSystem: "Windows, macOS, Linux",
            description: "CLI tool to scan, map, and re-clone Git repository trees.",
          }),
        }}
      />
    </DocsLayout>
  );
};

export default HomePage;
