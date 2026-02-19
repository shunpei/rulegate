"use client";

import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CitationList } from "./citation-list";
import type { AskResponse } from "@/lib/types";

interface AnswerCardProps {
  question: string;
  response: AskResponse;
}

export function AnswerCard({ question, response }: AnswerCardProps) {
  const isNotFound = response.confidence === 0 && response.citations.length === 0;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base font-medium">{question}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="whitespace-pre-wrap leading-relaxed">
          {response.answer_ja}
        </p>
        {!isNotFound && <CitationList citations={response.citations} />}
      </CardContent>
      <CardFooter className="flex items-center gap-2 text-xs text-muted-foreground">
        <Badge variant="outline" className="text-xs">
          confidence: {response.confidence.toFixed(2)}
        </Badge>
        <span>corpus: {response.meta.rag_corpus}</span>
      </CardFooter>
    </Card>
  );
}
