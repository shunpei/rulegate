export interface AskRequest {
  question_ja: string;
  discipline?: string;
  rule_edition?: string;
  context?: QueryContext;
  options?: RequestOption;
}

export interface QueryContext {
  boat_class?: string;
  race_phase?: string;
  notes?: string;
}

export interface RequestOption {
  top_k?: number;
  min_confidence?: number;
  return_contexts?: boolean;
  answer_style?: string;
}

export interface AskResponse {
  answer_ja: string;
  confidence: number;
  citations: Citation[];
  meta: Meta;
}

export interface Citation {
  rule_id: string;
  section_title: string;
  quote_en: string;
  source_url: string;
  score: number;
}

export interface Meta {
  rag_corpus: string;
  top_k: number;
  warnings: string[];
}

export interface ErrorResponse {
  error: string;
  code?: string;
  details?: string;
}
