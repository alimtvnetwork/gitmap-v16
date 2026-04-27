import { ReactNode } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DOCS_TOOLTIP_ALIGN,
  DOCS_TOOLTIP_SIDE,
  DOCS_TOOLTIP_SIDE_OFFSET,
} from "@/components/docs/docsTooltip";

interface DocsTooltipProps {
  // The control the tooltip describes. Pass the trigger element
  // directly; this wrapper hands it to TooltipTrigger asChild so
  // the trigger keeps its own ref/keyboard semantics.
  children: ReactNode;
  // Tooltip body. Keep it short (one short phrase) — long text
  // belongs in inline help, not a hover tooltip.
  label: ReactNode;
}

// DocsTooltip is the ONLY way to attach a hover tooltip in the
// docs header / toolbars. Centralizing here means every tooltip
// shares the same side, offset, and (via the provider in App.tsx)
// open/close delay. Do NOT inline a raw <Tooltip> in docs surfaces.
export const DocsTooltip = ({ children, label }: DocsTooltipProps) => (
  <Tooltip>
    <TooltipTrigger asChild>{children}</TooltipTrigger>
    <TooltipContent
      side={DOCS_TOOLTIP_SIDE}
      sideOffset={DOCS_TOOLTIP_SIDE_OFFSET}
      align={DOCS_TOOLTIP_ALIGN}
    >
      {label}
    </TooltipContent>
  </Tooltip>
);

export default DocsTooltip;
