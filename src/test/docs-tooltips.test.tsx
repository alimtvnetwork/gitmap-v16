import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/components/ui/sidebar";
import DocsLayout from "@/components/docs/DocsLayout";
import CodeBlock from "@/components/docs/CodeBlock";

// Why focus() instead of hover():
// Radix Tooltip opens on either pointer-enter OR focus. jsdom does
// not synthesize pointer events the way real browsers do, so
// focus is the deterministic path to "tooltip is open" in this
// environment. Real-browser pointer behaviour is covered by the
// shared DocsTooltip component (single source of truth for
// placement/delay) — this suite verifies WIRING, not Radix internals.

const renderDocsChrome = () =>
  render(
    <MemoryRouter>
      <TooltipProvider delayDuration={0} skipDelayDuration={0}>
        <SidebarProvider>
          <DocsLayout>
            <div>content</div>
          </DocsLayout>
        </SidebarProvider>
      </TooltipProvider>
    </MemoryRouter>,
  );

const renderCodeBlock = () =>
  render(
    <TooltipProvider delayDuration={0} skipDelayDuration={0}>
      <CodeBlock code={"line one\nline two"} language="bash" />
    </TooltipProvider>,
  );

// HEADER_CONTROLS lists every icon-only control in the docs header.
// Each entry pairs the trigger's accessible name (used to look the
// element up via getByRole / getByLabelText) with the substring we
// expect to find in the surfaced tooltip body. Update this list
// whenever a control is added/removed from the header.
const HEADER_CONTROLS: Array<{ trigger: string; tooltip: string }> = [
  { trigger: "Toggle sidebar", tooltip: "Toggle sidebar" },
  { trigger: "Dark theme", tooltip: "Dark theme" },
  { trigger: "Light theme", tooltip: "Light theme" },
  { trigger: /Copy .* theme palette to clipboard/i.source, tooltip: "palette" },
  {
    trigger: "Open command palette (search commands, flags, pages)",
    tooltip: "Search commands",
  },
];

// CODEBLOCK_CONTROLS lists every icon-only control in the CodeBlock
// toolbar. Same shape and update rule as HEADER_CONTROLS.
const CODEBLOCK_CONTROLS: Array<{ trigger: string; tooltip: string }> = [
  { trigger: "Decrease font size", tooltip: "Decrease font size" },
  { trigger: "Increase font size", tooltip: "Increase font size" },
  { trigger: "Copy snippet", tooltip: "Copy snippet" },
  { trigger: "Download snippet", tooltip: "Download snippet" },
  { trigger: "Enter fullscreen", tooltip: "Fullscreen" },
];

const findTriggerByName = (name: string) => {
  // Try exact aria-label first; fall back to a regex match for the
  // dynamic CopyPaletteButton label that includes the active theme.
  try {
    return screen.getByLabelText(name);
  } catch {
    return screen.getByLabelText(new RegExp(name, "i"));
  }
};

const expectTooltipFor = async (
  triggerName: string,
  tooltipSubstr: string,
) => {
  const trigger = findTriggerByName(triggerName);
  // Focus is the jsdom-friendly opener (see top-of-file comment).
  trigger.focus();
  // Radix mounts the tooltip body in a portal as role="tooltip".
  // findAllByRole tolerates same-text matches across header + portal.
  const tips = await screen.findAllByRole("tooltip");
  const matched = tips.some((t) =>
    (t.textContent ?? "").toLowerCase().includes(tooltipSubstr.toLowerCase()),
  );
  expect(
    matched,
    `expected an open tooltip whose text includes "${tooltipSubstr}" ` +
      `for trigger "${triggerName}", got: ${tips.map((t) => t.textContent).join(" | ")}`,
  ).toBe(true);
  trigger.blur();
};

describe("docs tooltip wiring — header controls", () => {
  beforeEach(() => cleanup());

  for (const ctl of HEADER_CONTROLS) {
    it(`shows tooltip "${ctl.tooltip}" for "${ctl.trigger}"`, async () => {
      renderDocsChrome();
      await expectTooltipFor(ctl.trigger, ctl.tooltip);
    });
  }
});

describe("docs tooltip wiring — CodeBlock toolbar", () => {
  beforeEach(() => cleanup());

  for (const ctl of CODEBLOCK_CONTROLS) {
    it(`shows tooltip "${ctl.tooltip}" for "${ctl.trigger}"`, async () => {
      renderCodeBlock();
      await expectTooltipFor(ctl.trigger, ctl.tooltip);
    });
  }
});

describe("docs tooltip wiring — version badge (non-icon trigger)", () => {
  beforeEach(() => cleanup());

  it("surfaces a tooltip for the version badge on focus", async () => {
    renderDocsChrome();
    // The version badge has aria-label="gitmap version <X>" (no icon
    // but it is a focus-able decorative chip — the audit lists it
    // as needing a tooltip alongside the icon-only controls).
    const badge = screen.getByLabelText(/gitmap version /i);
    badge.focus();
    const tips = await screen.findAllByRole("tooltip");
    const matched = tips.some((t) =>
      (t.textContent ?? "").toLowerCase().includes("gitmap version"),
    );
    expect(matched).toBe(true);
  });
});

// Pointer-event sanity check: also verify userEvent.hover() opens
// the tooltip in jsdom for at least one well-known trigger. If
// Radix changes its opener semantics this test will catch it.
describe("docs tooltip wiring — hover opener", () => {
  beforeEach(() => cleanup());

  it("opens the sidebar-toggle tooltip on pointer hover", async () => {
    const user = userEvent.setup();
    renderDocsChrome();
    const trigger = screen.getByLabelText("Toggle sidebar");
    await user.hover(trigger);
    const tips = await screen.findAllByRole("tooltip");
    const matched = tips.some((t) =>
      (t.textContent ?? "").toLowerCase().includes("toggle sidebar"),
    );
    expect(matched).toBe(true);
  });
});
