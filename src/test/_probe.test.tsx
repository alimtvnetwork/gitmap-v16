import { describe, it } from "vitest";
import { render } from "@testing-library/react";
import { TooltipProvider } from "@/components/ui/tooltip";
import { DocsTooltip } from "@/components/docs/DocsTooltip";

describe("probe", () => {
  it("string child", () => {
    const { container } = render(
      <TooltipProvider delayDuration={0}>
        <DocsTooltip label="my label">just text</DocsTooltip>
      </TooltipProvider>,
    );
    console.log("STRING:", container.innerHTML);
  });
  it("fragment child", () => {
    const { container } = render(
      <TooltipProvider delayDuration={0}>
        <DocsTooltip label="my label">
          <><span>a</span><span>b</span></>
        </DocsTooltip>
      </TooltipProvider>,
    );
    console.log("FRAG:", container.innerHTML);
  });
});
