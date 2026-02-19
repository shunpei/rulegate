"use client";

import { useState } from "react";
import { Header } from "@/components/header";
import { QuestionForm } from "@/components/question-form";
import { AnswerCard } from "@/components/answer-card";
import { askQuestion, ApiError } from "@/lib/apiClient";
import type { AskResponse } from "@/lib/types";

interface QAEntry {
  question: string;
  response: AskResponse;
}

export default function Home() {
  const [entries, setEntries] = useState<QAEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleAsk(question: string) {
    setIsLoading(true);
    setError(null);

    try {
      const response = await askQuestion({ question_ja: question });
      setEntries((prev) => [{ question, response }, ...prev]);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.body.error);
      } else {
        setError("サーバーとの通信に失敗しました。");
      }
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <main className="mx-auto max-w-3xl px-4 py-8">
        <div className="space-y-6">
          <QuestionForm onSubmit={handleAsk} isLoading={isLoading} />

          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
              {error}
            </div>
          )}

          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              <span className="ml-3 text-sm text-muted-foreground">
                回答を生成中...
              </span>
            </div>
          )}

          <div className="space-y-4">
            {entries.map((entry, i) => (
              <AnswerCard
                key={i}
                question={entry.question}
                response={entry.response}
              />
            ))}
          </div>
        </div>
      </main>
    </div>
  );
}
