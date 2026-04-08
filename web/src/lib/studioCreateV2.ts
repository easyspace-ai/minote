/**
 * Studio Create V2 - 后端主导的生成 API
 * 前端只需传 type + content + options，后端负责构建提示词和生成
 */

import { chatclawApi } from "@/api/chatclaw";

export type StudioCreateType =
  | "html"
  | "ppt"
  | "audio"
  | "mindmap"
  | "infographic"
  | "quiz"
  | "data_table";

export interface StudioCreateOptions {
  // HTML 选项
  page_type?: "article" | "report" | "landing" | "slides" | "dashboard" | "interactive";
  theme?: string;
  interactive?: boolean;

  // PPT 选项
  slide_count?: number;
  include_notes?: boolean;

  // Audio 选项
  language?: "zh-CN" | "en-US";
  voice?: string;
  speed?: number;

  // Infographic 选项
  layout?: "data" | "process" | "comparison" | "timeline" | "hierarchy";
  width?: number;
  height?: number;
  format?: "svg" | "png";

  // Quiz 选项
  question_types?: ("choice" | "true_false" | "fill_blank" | "short_answer")[];
  question_count?: number;
  difficulty?: "easy" | "medium" | "hard" | "mixed";
  show_answers?: boolean;

  // DataTable 选项
  table_type?: "comparison" | "list" | "matrix" | "timeline";
  sortable?: boolean;
  export_formats?: ("html" | "csv" | "xlsx")[];

  // Allow additional properties
  [key: string]: unknown;
}

export interface StudioCreateResult {
  success: boolean;
  material_id?: number;
  status: string;
  job_id?: string;
  error?: string;
  result?: {
    type: string;
    title: string;
    message?: string;
    theme?: string;
    slide_count?: number;
    download_url?: string;
    [key: string]: unknown;
  };
}

export interface StudioSkillInfo {
  type: string;
  name: string;
  version: string;
  description: string;
  trigger_keywords: string[];
  input_schema: {
    required: string[];
    optional?: string[];
  };
  output_schema: {
    fields: string[];
  };
  theme_options?: Record<string, unknown>;
  voice_options?: Record<string, unknown>;
  limits?: Record<string, unknown>;
}

/**
 * 创建 Studio 生成任务（V2 后端主导 API）
 */
export async function studioCreateV2(
  projectId: string,
  type: StudioCreateType,
  content: string,
  title?: string,
  options?: StudioCreateOptions,
  materialId?: number
): Promise<StudioCreateResult> {
  const response = await chatclawApi.projects.materials.studioCreate(projectId, {
    type,
    content,
    title: title || "未命名",
    options: (options || {}) as Record<string, unknown>,
    material_id: materialId,
  });

  return response as StudioCreateResult;
}

/**
 * 获取可用的 Studio Skills 列表
 */
export async function getStudioSkills(): Promise<StudioSkillInfo[]> {
  return chatclawApi.projects.materials.studioSkills();
}

/**
 * 根据关键词匹配最佳 Skill 类型
 */
export function matchStudioTypeByKeywords(input: string): StudioCreateType | null {
  const lower = input.toLowerCase();

  const patterns: { type: StudioCreateType; keywords: string[] }[] = [
    { type: "ppt", keywords: ["ppt", "幻灯片", "演示", "presentation", "slides", "deck"] },
    { type: "html", keywords: ["html", "网页", "页面", "web", "website", "landing"] },
    { type: "audio", keywords: ["audio", "音频", "语音", "tts", "播客", "podcast", "朗读"] },
    { type: "mindmap", keywords: ["mindmap", "思维导图", "脑图", "思维"] },
    { type: "infographic", keywords: ["infographic", "信息图", "可视化", "图表", "流程图"] },
    { type: "quiz", keywords: ["quiz", "测验", "考试", "题目", "练习", "测试"] },
    { type: "data_table", keywords: ["table", "表格", "数据", "spreadsheet", "excel"] },
  ];

  for (const { type, keywords } of patterns) {
    if (keywords.some((k) => lower.includes(k))) {
      return type;
    }
  }

  return null;
}

/**
 * 获取 Skill 的默认选项
 */
export function getDefaultOptions(type: StudioCreateType): StudioCreateOptions {
  switch (type) {
    case "html":
      return {
        page_type: "article",
        theme: "light",
        interactive: true,
      };
    case "ppt":
      return {
        theme: "professional",
        slide_count: 10,
        include_notes: true,
      };
    case "audio":
      return {
        language: "zh-CN",
        voice: "zh-female-1",
        speed: 1.0,
      };
    case "infographic":
      return {
        layout: "data",
        theme: "modern",
        format: "svg",
      };
    case "quiz":
      return {
        question_count: 10,
        difficulty: "mixed",
        show_answers: true,
      };
    case "data_table":
      return {
        table_type: "comparison",
        sortable: true,
        export_formats: ["html", "csv"],
      };
    default:
      return {};
  }
}

/**
 * 构建用户友好的选项描述
 */
export function buildOptionsDescription(
  type: StudioCreateType,
  options: StudioCreateOptions
): string {
  const parts: string[] = [];

  switch (type) {
    case "html":
      if (options.page_type) parts.push(`页面类型: ${options.page_type}`);
      if (options.theme) parts.push(`主题: ${options.theme}`);
      break;
    case "ppt":
      if (options.theme) parts.push(`主题: ${options.theme}`);
      if (options.slide_count) parts.push(`${options.slide_count} 页`);
      break;
    case "audio":
      if (options.language) parts.push(`语言: ${options.language}`);
      if (options.voice) parts.push(`音色: ${options.voice}`);
      break;
    case "infographic":
      if (options.layout) parts.push(`布局: ${options.layout}`);
      break;
    case "quiz":
      if (options.question_count) parts.push(`${options.question_count} 题`);
      if (options.difficulty) parts.push(`难度: ${options.difficulty}`);
      break;
  }

  return parts.join(" · ") || "默认配置";
}
