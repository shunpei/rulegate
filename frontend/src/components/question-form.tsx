"use client";

import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";

interface QuestionFormProps {
  onSubmit: (question: string) => void;
  isLoading: boolean;
}

export function QuestionForm({ onSubmit, isLoading }: QuestionFormProps) {
  const [question, setQuestion] = useState("");

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = question.trim();
    if (!trimmed) return;
    onSubmit(trimmed);
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <Textarea
        placeholder="競技規則について質問してください（例：ゲートに触った場合のペナルティは？）"
        value={question}
        onChange={(e) => setQuestion(e.target.value)}
        rows={3}
        maxLength={1000}
        disabled={isLoading}
        className="resize-none"
      />
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">
          {question.length}/1000
        </span>
        <Button type="submit" disabled={isLoading || !question.trim()}>
          {isLoading ? "回答を生成中..." : "質問する"}
        </Button>
      </div>
    </form>
  );
}
