import { parse, stringify } from "yaml";

export interface FrontmatterResult<T> {
  frontmatter: T;
  body: string;
}

export function parseFrontmatter<T extends Record<string, unknown>>(
  content: string,
): FrontmatterResult<T> | null {
  const match = content.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
  if (!match) {
    return null;
  }

  try {
    const frontmatter = parse(match[1], {
      strict: false,
      uniqueKeys: false,
    }) as T;
    const body = match[2];
    return { frontmatter, body };
  } catch (error) {
    console.error("Failed to parse frontmatter:", error);
    return parseFrontmatterRegex<T>(match[1], match[2]);
  }
}

function parseFrontmatterRegex<T extends Record<string, unknown>>(
  yamlStr: string,
  body: string,
): FrontmatterResult<T> | null {
  const result: Record<string, unknown> = {};
  const lines = yamlStr.split("\n");
  let currentKey: string | null = null;
  let currentValue: string[] = [];
  let inMultiline = false;

  for (const line of lines) {
    const keyMatch = line.match(/^(\w+):\s*(.*)$/);
    if (keyMatch && !line.startsWith(" ") && !line.startsWith("\t")) {
      if (currentKey && inMultiline) {
        result[currentKey] = currentValue.join(" ").trim();
      }
      currentKey = keyMatch[1];
      const value = keyMatch[2].trim();
      if (value) {
        if (value.startsWith("|") || value.startsWith(">")) {
          inMultiline = true;
          currentValue = [];
        } else {
          result[currentKey] = value;
          inMultiline = false;
        }
      } else {
        inMultiline = true;
        currentValue = [];
      }
    } else if (currentKey && inMultiline) {
      currentValue.push(line.trim());
    }
  }

  if (currentKey && inMultiline) {
    result[currentKey] = currentValue.join(" ").trim();
  }

  return { frontmatter: result as T, body };
}

export function stringifyFrontmatter<T extends Record<string, unknown>>(
  frontmatter: T,
  body: string,
): string {
  const yamlStr = stringify(frontmatter, {
    defaultStringType: "QUOTE_DOUBLE",
    defaultKeyType: "PLAIN",
    lineWidth: 0,
  });
  return `---\n${yamlStr}---\n${body}`;
}

export function extractYamlValue(content: string, key: string): string | null {
  const match = content.match(new RegExp(`^${key}:\\s*(.+)$`, "m"));
  return match ? match[1].trim() : null;
}
