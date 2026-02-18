import express from "express";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { z } from "zod";
import { TaskAIClient, Task, Project, SwimLane, Comment, WikiPage, WikiBlock } from "./api.js";

const TASKAI_API_URL = process.env.TASKAI_API_URL || "https://taskai.cc";
const PORT = parseInt(process.env.PORT || "3000", 10);

/**
 * Helper to format response with minimal tokens by default.
 * Use verbose=true to get full details with pretty formatting.
 */
function formatResponse(data: unknown, verbose = false): string {
  return verbose ? JSON.stringify(data, null, 2) : JSON.stringify(data);
}

/**
 * Extract minimal fields from a task for list operations.
 */
function minimizeTask(task: Task) {
  return {
    id: task.id,
    task_number: task.task_number,
    title: task.title,
    status: task.status,
    priority: task.priority,
  };
}

/**
 * Extract minimal fields from a project for list operations.
 */
function minimizeProject(project: Project) {
  return {
    id: project.id,
    name: project.name,
  };
}

/**
 * Extract minimal fields from a swim lane.
 */
function minimizeSwimLane(lane: SwimLane) {
  return {
    id: lane.id,
    name: lane.name,
    status_category: lane.status_category,
  };
}

/**
 * Extract minimal fields from a comment.
 */
function minimizeComment(comment: Comment) {
  return {
    id: comment.id,
    content: comment.content,
    author_id: comment.author_id,
    created_at: comment.created_at,
  };
}

/**
 * Create and configure the MCP server with all TaskAI tools.
 */
function createServer(client: TaskAIClient): McpServer {
  const server = new McpServer({
    name: "taskai",
    version: "1.0.0",
  });

  // --- get_me ---
  server.tool(
    "get_me",
    "Get current authenticated user info",
    { verbose: z.boolean().optional().describe("Return full details (default: false)") },
    async ({ verbose }) => {
      const user = await client.getMe();
      return { content: [{ type: "text", text: formatResponse(user, verbose) }] };
    }
  );

  // --- list_projects ---
  server.tool(
    "list_projects",
    "List all projects (minimal fields by default, use verbose=true for full details)",
    {
      page: z.number().optional(),
      limit: z.number().optional(),
      verbose: z.boolean().optional().describe("Return full details (default: false)"),
    },
    async ({ page, limit, verbose }) => {
      const result = await client.listProjects(page, limit);
      const data = verbose
        ? result
        : { projects: result.projects.map(minimizeProject), total: result.total };
      return { content: [{ type: "text", text: formatResponse(data, verbose) }] };
    }
  );

  // --- get_project ---
  server.tool(
    "get_project",
    "Get project details by ID",
    {
      project_id: z.string().describe("Project ID"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ project_id, verbose }) => {
      const project = await client.getProject(project_id);
      return { content: [{ type: "text", text: formatResponse(project, verbose) }] };
    }
  );

  // --- list_swim_lanes ---
  server.tool(
    "list_swim_lanes",
    "List swim lanes (columns) for a project. Each lane has a status_category (todo, in_progress, done) that determines task status. Returns minimal fields by default.",
    {
      project_id: z.string().describe("Project ID"),
      verbose: z.boolean().optional().describe("Return full details (default: false)"),
    },
    async ({ project_id, verbose }) => {
      const lanes = await client.listSwimLanes(project_id);
      const data = verbose ? lanes : lanes.map(minimizeSwimLane);
      return { content: [{ type: "text", text: formatResponse(data, verbose) }] };
    }
  );

  // --- list_tasks ---
  server.tool(
    "list_tasks",
    "List tasks in a project (optional status/search filter). Returns minimal fields by default (id, task_number, title, status, priority). Use verbose=true for full task details.",
    {
      project_id: z.string().describe("Project ID"),
      query: z.string().optional().describe("Search query"),
      status: z.string().optional().describe("Filter by status (e.g. todo, in_progress, done)"),
      page: z.number().optional(),
      limit: z.number().optional(),
      verbose: z.boolean().optional().describe("Return full task details (default: false)"),
    },
    async ({ project_id, query, status, page, limit, verbose }) => {
      const result = await client.listTasks(project_id, { query, status, page, limit });
      const data = verbose
        ? result
        : { tasks: result.tasks.map(minimizeTask), total: result.total };
      return { content: [{ type: "text", text: formatResponse(data, verbose) }] };
    }
  );

  // --- get_task ---
  server.tool(
    "get_task",
    "Get a single task by its project-scoped task number",
    {
      project_id: z.string().describe("Project ID"),
      task_number: z.number().describe("Task number within the project (e.g. 1, 2, 3)"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ project_id, task_number, verbose }) => {
      const task = await client.getTaskByNumber(project_id, task_number);
      return { content: [{ type: "text", text: formatResponse(task, verbose) }] };
    }
  );

  // --- create_task ---
  server.tool(
    "create_task",
    "Create a new task in a project",
    {
      project_id: z.string().describe("Project ID"),
      title: z.string().describe("Task title"),
      description: z.string().optional().describe("Task description"),
      status: z.string().optional().describe("Task status (default: todo)"),
      priority: z.string().optional().describe("Priority: low, medium, high, critical"),
      assigned_to: z.string().optional().describe("User ID to assign"),
      swim_lane_id: z.number().optional().describe("Swim lane ID (use list_swim_lanes to get valid IDs)"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ project_id, title, description, status, priority, assigned_to, swim_lane_id, verbose }) => {
      const task = await client.createTask(project_id, { title, description, status, priority, assigned_to, swim_lane_id });
      return { content: [{ type: "text", text: formatResponse(task, verbose) }] };
    }
  );

  // --- update_task ---
  server.tool(
    "update_task",
    "Update an existing task",
    {
      task_id: z.string().describe("Task ID"),
      title: z.string().optional().describe("New title"),
      description: z.string().optional().describe("New description"),
      status: z.string().optional().describe("New status"),
      priority: z.string().optional().describe("New priority"),
      assigned_to: z.string().optional().describe("New assignee user ID"),
      swim_lane_id: z.number().optional().describe("Swim lane ID (use list_swim_lanes to get valid IDs)"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ task_id, title, description, status, priority, assigned_to, swim_lane_id, verbose }) => {
      const task = await client.updateTask(task_id, { title, description, status, priority, assigned_to, swim_lane_id });
      return { content: [{ type: "text", text: formatResponse(task, verbose) }] };
    }
  );

  // --- list_comments ---
  server.tool(
    "list_comments",
    "List comments on a task. Returns minimal fields by default.",
    {
      task_id: z.string().describe("Task ID"),
      verbose: z.boolean().optional().describe("Return full details (default: false)"),
    },
    async ({ task_id, verbose }) => {
      const result = await client.listComments(task_id);
      const data = verbose
        ? result
        : { comments: result.comments.map(minimizeComment) };
      return { content: [{ type: "text", text: formatResponse(data, verbose) }] };
    }
  );

  // --- add_comment ---
  server.tool(
    "add_comment",
    "Add a comment to a task",
    {
      task_id: z.string().describe("Task ID"),
      content: z.string().describe("Comment text"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ task_id, content, verbose }) => {
      const comment = await client.addComment(task_id, content);
      return { content: [{ type: "text", text: formatResponse(comment, verbose) }] };
    }
  );

  // --- search_wiki ---
  server.tool(
    "search_wiki",
    "Search wiki content across pages (full-text search)",
    {
      query: z.string().describe("Search query"),
      project_id: z.string().optional().describe("Filter by project ID"),
      limit: z.number().optional().describe("Max results (default: 20, max: 100)"),
      recency_days: z.number().optional().describe("Only return pages updated in last N days"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ query, project_id, limit, recency_days, verbose }) => {
      const result = await client.searchWiki({ query, project_id, limit, recency_days });
      return { content: [{ type: "text", text: formatResponse(result, verbose) }] };
    }
  );

  // --- list_wiki_pages ---
  server.tool(
    "list_wiki_pages",
    "List all wiki pages in a project",
    {
      project_id: z.string().describe("Project ID"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ project_id, verbose }) => {
      const pages = await client.listWikiPages(project_id);
      return { content: [{ type: "text", text: formatResponse(pages, verbose) }] };
    }
  );

  // --- get_wiki_page ---
  server.tool(
    "get_wiki_page",
    "Get a specific wiki page by ID",
    {
      page_id: z.string().describe("Wiki page ID"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ page_id, verbose }) => {
      const page = await client.getWikiPage(page_id);
      return { content: [{ type: "text", text: formatResponse(page, verbose) }] };
    }
  );

  // --- autocomplete_wiki_pages ---
  server.tool(
    "autocomplete_wiki_pages",
    "Autocomplete wiki page titles (fuzzy search)",
    {
      query: z.string().describe("Search query for page title"),
      project_id: z.string().optional().describe("Filter by project ID"),
      limit: z.number().optional().describe("Max results (default: 10, max: 50)"),
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ query, project_id, limit, verbose }) => {
      const results = await client.autocompletePages(query, project_id, limit);
      return { content: [{ type: "text", text: formatResponse(results, verbose) }] };
    }
  );

  // --- get_version ---
  server.tool(
    "get_version",
    "Get system version information (backend version, DB migration version, build info)",
    {
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ verbose }) => {
      const version = await client.getVersion();
      return { content: [{ type: "text", text: formatResponse(version, verbose) }] };
    }
  );

  // --- health_check ---
  server.tool(
    "health_check",
    "Check system health status (database connectivity)",
    {
      verbose: z.boolean().optional().describe("Pretty print JSON (default: false)"),
    },
    async ({ verbose }) => {
      const health = await client.healthCheck();
      return { content: [{ type: "text", text: formatResponse(health, verbose) }] };
    }
  );

  return server;
}

// --- Express app ---
const app = express();
app.use(express.json());

// Health endpoint
app.get("/health", (_req, res) => {
  res.json({ status: "ok", service: "taskai-mcp" });
});

// MCP endpoint — stateless: one transport per request
app.post("/mcp", async (req, res) => {
  // Extract API key from X-API-Key header
  const apiKey = req.headers["x-api-key"] as string | undefined;
  if (!apiKey) {
    res.status(401).json({ error: "Missing X-API-Key header" });
    return;
  }

  // Validate the API key by calling /api/me
  const client = new TaskAIClient(TASKAI_API_URL, apiKey);
  try {
    await client.getMe();
  } catch {
    res.status(403).json({ error: "Invalid API key" });
    return;
  }

  // Create MCP server with authenticated client
  const server = createServer(client);

  // Stateless transport — no session persistence
  const transport = new StreamableHTTPServerTransport({
    sessionIdGenerator: undefined,
  });
  await server.connect(transport);
  await transport.handleRequest(req, res, req.body);
});

// Handle GET and DELETE on /mcp for protocol compliance (stateless = 405)
app.get("/mcp", (_req, res) => {
  res.status(405).json({ error: "Method not allowed — stateless server, use POST" });
});

app.delete("/mcp", (_req, res) => {
  res.status(405).json({ error: "Method not allowed — stateless server, use POST" });
});

app.listen(PORT, () => {
  console.log(`TaskAI MCP server listening on port ${PORT}`);
  console.log(`API backend: ${TASKAI_API_URL}`);
});
