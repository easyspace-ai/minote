import type { Message } from "@langchain/langgraph-sdk";
import {
  BookOpenTextIcon,
  ChevronUp,
  FolderOpenIcon,
  GlobeIcon,
  LightbulbIcon,
  ListTodoIcon,
  MessageCircleQuestionMarkIcon,
  NotebookPenIcon,
  SearchIcon,
  SquareTerminalIcon,
  WrenchIcon,
} from "lucide-react";
import { useMemo, useState } from "react";

import {
  ChainOfThought,
  ChainOfThoughtContent,
  ChainOfThoughtSearchResult,
  ChainOfThoughtSearchResults,
  ChainOfThoughtStep,
} from "@/components/ai-elements/chain-of-thought";
import { CodeBlock } from "@/components/ai-elements/code-block";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/core/i18n/hooks";
import {
  extractReasoningContentFromMessage,
  findToolCallResult,
} from "@/core/messages/utils";
import { useRehypeSplitWordsIntoSpans } from "@/core/rehype";
import { extractTitleFromMarkdown } from "@/core/utils/markdown";
import { env } from "@/env";
import { cn } from "@/lib/utils";

import { FlipDisplay } from "../flip-display";
import { Tooltip } from "../tooltip";

import { MarkdownContent } from "./markdown-content";

export function MessageGroup({
  className,
  messages,
  isLoading = false,
}: {
  className?: string;
  messages: Message[];
  isLoading?: boolean;
}) {
  const { t } = useI18n();
  const [showAbove, setShowAbove] = useState(
    env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true",
  );
  const [showLastThinking, setShowLastThinking] = useState(
    env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true",
  );
  const steps = useMemo(() => convertToSteps(messages), [messages]);
  const lastToolCallStep = useMemo(() => {
    const filteredSteps = steps.filter((step) => step.type === "toolCall");
    return filteredSteps[filteredSteps.length - 1];
  }, [steps]);
  const aboveLastToolCallSteps = useMemo(() => {
    if (lastToolCallStep) {
      const index = steps.indexOf(lastToolCallStep);
      return steps.slice(0, index);
    }
    return [];
  }, [lastToolCallStep, steps]);
  const lastReasoningStep = useMemo(() => {
    if (lastToolCallStep) {
      const index = steps.indexOf(lastToolCallStep);
      return steps.slice(index + 1).find((step) => step.type === "reasoning");
    } else {
      const filteredSteps = steps.filter((step) => step.type === "reasoning");
      return filteredSteps[filteredSteps.length - 1];
    }
  }, [lastToolCallStep, steps]);
  const rehypePlugins = useRehypeSplitWordsIntoSpans(isLoading);
  return (
    <ChainOfThought
      className={cn("w-full gap-2 rounded-lg border p-0.5", className)}
      open={true}
    >
      {aboveLastToolCallSteps.length > 0 && (
        <Button
          key="above"
          className="w-full items-start justify-start text-left"
          variant="ghost"
          onClick={() => setShowAbove(!showAbove)}
        >
          <ChainOfThoughtStep
            label={
              <span className="opacity-60">
                {showAbove
                  ? t.toolCalls.lessSteps
                  : t.toolCalls.moreSteps(aboveLastToolCallSteps.length)}
              </span>
            }
            icon={
              <ChevronUp
                className={cn(
                  "size-4 opacity-60 transition-transform duration-200",
                  showAbove ? "rotate-180" : "",
                )}
              />
            }
          ></ChainOfThoughtStep>
        </Button>
      )}
      {lastToolCallStep && (
        <ChainOfThoughtContent className="px-4 pb-2">
          {showAbove &&
            aboveLastToolCallSteps.map((step, index) =>
              step.type === "reasoning" ? (
                <ChainOfThoughtStep
                  key={stableStepKey(step, index)}
                  label={
                    <MarkdownContent
                      content={step.reasoning ?? ""}
                      isLoading={isLoading}
                      rehypePlugins={rehypePlugins}
                    />
                  }
                ></ChainOfThoughtStep>
              ) : (
                <ToolCall
                  key={stableStepKey(step, index)}
                  {...step}
                  isLoading={isLoading}
                />
              ),
            )}
          {lastToolCallStep && (
            <FlipDisplay
              uniqueKey={stableStepKey(lastToolCallStep, "last")}
            >
              <ToolCall
                key={stableStepKey(lastToolCallStep, "last")}
                {...lastToolCallStep}
                isLast={true}
                isLoading={isLoading}
              />
            </FlipDisplay>
          )}
        </ChainOfThoughtContent>
      )}
      {lastReasoningStep && (
        <>
          <Button
            key={stableStepKey(lastReasoningStep, "reasoning-btn")}
            className="w-full items-start justify-start text-left"
            variant="ghost"
            onClick={() => setShowLastThinking(!showLastThinking)}
          >
            <div className="flex w-full items-center justify-between">
              <ChainOfThoughtStep
                className="font-normal"
                label={t.common.thinking}
                icon={LightbulbIcon}
              ></ChainOfThoughtStep>
              <div>
                <ChevronUp
                  className={cn(
                    "text-muted-foreground size-4",
                    showLastThinking ? "" : "rotate-180",
                  )}
                />
              </div>
            </div>
          </Button>
          {showLastThinking && (
            <ChainOfThoughtContent className="px-4 pb-2">
              <ChainOfThoughtStep
                key={stableStepKey(lastReasoningStep, "reasoning-content")}
                label={
                  <MarkdownContent
                    content={lastReasoningStep.reasoning ?? ""}
                    isLoading={isLoading}
                    rehypePlugins={rehypePlugins}
                  />
                }
              ></ChainOfThoughtStep>
            </ChainOfThoughtContent>
          )}
        </>
      )}
    </ChainOfThought>
  );
}

function ToolCall({
  id,
  messageId,
  name,
  args: argsMaybe,
  result,
  isLast = false,
  isLoading = false,
}: {
  id?: string;
  messageId?: string;
  name: string;
  args?: Record<string, unknown> | null;
  result?: string | Record<string, unknown>;
  isLast?: boolean;
  isLoading?: boolean;
}) {
  const args = argsMaybe ?? {};
  const { t } = useI18n();

  if (name === "web_search") {
    let label: React.ReactNode = t.toolCalls.searchForRelatedInfo;
    if (typeof args.query === "string") {
      label = t.toolCalls.searchOnWebFor(args.query);
    }
    return (
      <ChainOfThoughtStep key={id} label={label} icon={SearchIcon}>
        {Array.isArray(result) && (
          <ChainOfThoughtSearchResults>
            {result.map((item) => (
              <ChainOfThoughtSearchResult key={item.url}>
                <a href={item.url} target="_blank" rel="noopener noreferrer">
                  {item.title}
                </a>
              </ChainOfThoughtSearchResult>
            ))}
          </ChainOfThoughtSearchResults>
        )}
      </ChainOfThoughtStep>
    );
  } else if (name === "image_search") {
    let label: React.ReactNode = t.toolCalls.searchForRelatedImages;
    if (typeof args.query === "string") {
      label = t.toolCalls.searchForRelatedImagesFor(args.query);
    }
    const results = (
      result as {
        results: {
          source_url: string;
          thumbnail_url: string;
          image_url: string;
          title: string;
        }[];
      }
    )?.results;
    return (
      <ChainOfThoughtStep key={id} label={label} icon={SearchIcon}>
        {Array.isArray(results) && (
          <ChainOfThoughtSearchResults>
            {Array.isArray(results) &&
              results.map((item) => (
                <Tooltip key={item.image_url} content={item.title}>
                  <a
                    className="size-24 overflow-hidden rounded-lg object-cover"
                    href={item.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <div className="bg-accent size-24">
                      <img
                        className="size-full object-cover"
                        src={item.thumbnail_url}
                        alt={item.title}
                        width={100}
                        height={100}
                      />
                    </div>
                  </a>
                </Tooltip>
              ))}
          </ChainOfThoughtSearchResults>
        )}
      </ChainOfThoughtStep>
    );
  } else if (name === "web_fetch") {
    const url = (args as { url: string })?.url;
    let title = url;
    if (typeof result === "string") {
      const potentialTitle = extractTitleFromMarkdown(result);
      if (potentialTitle && potentialTitle.toLowerCase() !== "untitled") {
        title = potentialTitle;
      }
    }
    return (
      <ChainOfThoughtStep
        key={id}
        className="cursor-pointer"
        label={t.toolCalls.viewWebPage}
        icon={GlobeIcon}
        onClick={() => {
          window.open(url, "_blank");
        }}
      >
        <ChainOfThoughtSearchResult>
          {url && (
            <a href={url} target="_blank" rel="noopener noreferrer">
              {title}
            </a>
          )}
        </ChainOfThoughtSearchResult>
      </ChainOfThoughtStep>
    );
  } else if (name === "ls") {
    let description: string | undefined = (args as { description: string })
      ?.description;
    if (!description) {
      description = t.toolCalls.listFolder;
    }
    const path: string | undefined = (args as { path: string })?.path;
    return (
      <ChainOfThoughtStep key={id} label={description} icon={FolderOpenIcon}>
        {path && (
          <ChainOfThoughtSearchResult className="cursor-pointer">
            {path}
          </ChainOfThoughtSearchResult>
        )}
      </ChainOfThoughtStep>
    );
  } else if (name === "read_file") {
    let description: string | undefined = (args as { description: string })
      ?.description;
    if (!description) {
      description = t.toolCalls.readFile;
    }
    const path = (args as { path?: string; content?: string })?.path;
    return (
      <ChainOfThoughtStep key={id} label={description} icon={BookOpenTextIcon}>
        {path && (
          <ChainOfThoughtSearchResult className="cursor-pointer">
            {path}
          </ChainOfThoughtSearchResult>
        )}
      </ChainOfThoughtStep>
    );
  } else if (name === "write_file" || name === "str_replace") {
    let description: string | undefined = (args as { description: string })
      ?.description;
    if (!description) {
      description = t.toolCalls.writeFile;
    }
    const path: string | undefined = (args as { path: string })?.path;

    return (
      <ChainOfThoughtStep key={id} label={description} icon={NotebookPenIcon}>
        {path && (
          <ChainOfThoughtSearchResult>{path}</ChainOfThoughtSearchResult>
        )}
      </ChainOfThoughtStep>
    );
  } else if (name === "bash") {
    const description: string | undefined = (args as { description: string })
      ?.description;
    if (!description) {
      return t.toolCalls.executeCommand;
    }
    const command: string | undefined = (args as { command: string })?.command;
    return (
      <ChainOfThoughtStep
        key={id}
        label={description}
        icon={SquareTerminalIcon}
      >
        {command && (
          <CodeBlock
            className="mx-0 cursor-pointer border-none px-0"
            showLineNumbers={false}
            language="bash"
            code={command}
          />
        )}
      </ChainOfThoughtStep>
    );
  } else if (name === "ask_clarification") {
    return (
      <ChainOfThoughtStep
        key={id}
        label={t.toolCalls.needYourHelp}
        icon={MessageCircleQuestionMarkIcon}
      ></ChainOfThoughtStep>
    );
  } else if (name === "write_todos") {
    return (
      <ChainOfThoughtStep
        key={id}
        label={t.toolCalls.writeTodos}
        icon={ListTodoIcon}
      ></ChainOfThoughtStep>
    );
  } else {
    const description: string | undefined = (args as { description: string })
      ?.description;
    const displayName = name.trim() !== "" ? name : "unknown";
    return (
      <ChainOfThoughtStep
        key={id}
        label={description ?? t.toolCalls.useTool(displayName)}
        icon={WrenchIcon}
      ></ChainOfThoughtStep>
    );
  }
}

interface GenericCoTStep<T extends string = string> {
  id?: string;
  messageId?: string;
  type: T;
}

interface CoTReasoningStep extends GenericCoTStep<"reasoning"> {
  reasoning: string | null;
}

interface CoTToolCallStep extends GenericCoTStep<"toolCall"> {
  name: string;
  args: Record<string, unknown>;
  result?: string;
}

type CoTStep = CoTReasoningStep | CoTToolCallStep;

function stableStepKey(step: CoTStep, index: number | string): string {
  const id = step.id?.trim();
  const mid = step.messageId?.trim() ?? "";
  if (id) {
    return `${mid}-${id}`;
  }
  return `${mid}-${step.type}-${String(index)}`;
}

function convertToSteps(messages: Message[]): CoTStep[] {
  const steps: CoTStep[] = [];
  for (const message of messages) {
    if (message.type === "ai") {
      const reasoning = extractReasoningContentFromMessage(message);
      if (reasoning) {
        const step: CoTReasoningStep = {
          id: message.id,
          messageId: message.id,
          type: "reasoning",
          reasoning: extractReasoningContentFromMessage(message),
        };
        steps.push(step);
      }
      let toolIndex = 0;
      for (const tool_call of message.tool_calls ?? []) {
        if (tool_call.name === "task") {
          continue;
        }
        const rawId = tool_call.id?.trim();
        const syntheticId = rawId
          ? rawId
          : `${message.id ?? "msg"}-tc-${toolIndex}-${tool_call.name ?? "unnamed"}`;
        const step: CoTToolCallStep = {
          id: syntheticId,
          messageId: message.id,
          type: "toolCall",
          name: tool_call.name ?? "",
          args:
            tool_call.args != null && typeof tool_call.args === "object"
              ? (tool_call.args as Record<string, unknown>)
              : {},
        };
        toolIndex += 1;
        // Only the real API id can match tool messages' tool_call_id; synthetic ids are UI-only.
        if (rawId) {
          const toolCallResult = findToolCallResult(rawId, messages);
          if (toolCallResult) {
            try {
              const json = JSON.parse(toolCallResult);
              step.result = json;
            } catch {
              step.result = toolCallResult;
            }
          }
        }
        steps.push(step);
      }
    }
  }
  return steps;
}
