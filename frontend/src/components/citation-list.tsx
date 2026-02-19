import { Badge } from "@/components/ui/badge";
import type { Citation } from "@/lib/types";

interface CitationListProps {
  citations: Citation[];
}

export function CitationList({ citations }: CitationListProps) {
  if (citations.length === 0) return null;

  return (
    <div className="space-y-2">
      <h3 className="text-sm font-medium text-muted-foreground">
        引用 ({citations.length})
      </h3>
      <ul className="space-y-2">
        {citations.map((c, i) => (
          <li
            key={i}
            className="rounded-md border bg-muted/50 px-3 py-2 text-sm"
          >
            <div className="flex items-center gap-2">
              {c.rule_id && (
                <Badge variant="secondary" className="text-xs">
                  {c.rule_id}
                </Badge>
              )}
              {c.section_title && (
                <span className="text-xs text-muted-foreground">
                  {c.section_title}
                </span>
              )}
              <span className="ml-auto text-xs text-muted-foreground">
                score: {c.score.toFixed(2)}
              </span>
            </div>
            {c.quote_en && (
              <p className="mt-1 text-xs italic text-muted-foreground">
                &ldquo;{c.quote_en}&rdquo;
              </p>
            )}
            {c.source_url && (
              <a
                href={c.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-1 inline-block text-xs text-primary underline"
              >
                Source
              </a>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
