const express = require('express');
const Y = require('yjs');
const { Buffer } = require('buffer');

const app = express();
const PORT = process.env.PORT || 3001;

// Middleware
app.use(express.json({ limit: '10mb' }));

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'yjs-processor' });
});

/**
 * POST /apply-updates
 *
 * Applies an array of Yjs updates to a document and returns the current state.
 *
 * Request body:
 * {
 *   "updates": ["base64update1", "base64update2", ...]
 * }
 *
 * Response:
 * {
 *   "state": "base64encodedstate"
 * }
 */
app.post('/apply-updates', (req, res) => {
  try {
    const { updates } = req.body;

    if (!Array.isArray(updates)) {
      return res.status(400).json({
        error: 'updates must be an array of base64-encoded strings'
      });
    }

    // Create a new Yjs document
    const ydoc = new Y.Doc();

    // Apply all updates in order
    for (const updateBase64 of updates) {
      try {
        const updateBuffer = Buffer.from(updateBase64, 'base64');
        Y.applyUpdate(ydoc, new Uint8Array(updateBuffer));
      } catch (err) {
        return res.status(400).json({
          error: 'Invalid update format',
          details: err.message
        });
      }
    }

    // Encode the current state
    const state = Y.encodeStateAsUpdate(ydoc);
    const stateBase64 = Buffer.from(state).toString('base64');

    res.json({ state: stateBase64 });
  } catch (err) {
    console.error('Error applying updates:', err);
    res.status(500).json({
      error: 'Failed to apply updates',
      details: err.message
    });
  }
});

/**
 * POST /extract-blocks
 *
 * Extracts content blocks from a Yjs document state for indexing.
 *
 * Request body:
 * {
 *   "state": "base64encodedstate"
 * }
 *
 * Response:
 * {
 *   "blocks": [
 *     {
 *       "type": "paragraph",
 *       "level": null,
 *       "headings_path": "",
 *       "plain_text": "This is a paragraph",
 *       "canonical_json": "{\"type\":\"paragraph\",\"content\":[...]}"
 *     },
 *     {
 *       "type": "heading",
 *       "level": 1,
 *       "headings_path": "Introduction",
 *       "plain_text": "Introduction",
 *       "canonical_json": "{\"type\":\"heading\",\"attrs\":{\"level\":1},\"content\":[...]}"
 *     }
 *   ]
 * }
 */
app.post('/extract-blocks', (req, res) => {
  try {
    const { state } = req.body;

    if (!state || typeof state !== 'string') {
      return res.status(400).json({
        error: 'state must be a base64-encoded string'
      });
    }

    // Create a new Yjs document
    const ydoc = new Y.Doc();

    // Apply the state
    try {
      const stateBuffer = Buffer.from(state, 'base64');
      Y.applyUpdate(ydoc, new Uint8Array(stateBuffer));
    } catch (err) {
      return res.status(400).json({
        error: 'Invalid state format',
        details: err.message
      });
    }

    // Get the shared type (assuming ProseMirror uses 'default' or 'prosemirror')
    // Try common shared type names
    const sharedType = ydoc.get('default', Y.XmlFragment) ||
                      ydoc.get('prosemirror', Y.XmlFragment) ||
                      ydoc.get('content', Y.XmlFragment);

    if (!sharedType) {
      return res.json({ blocks: [] });
    }

    // Extract blocks from the shared type
    const blocks = [];
    const headingsPath = [];

    function extractFromNode(node, position = 0) {
      if (!node) return position;

      if (node instanceof Y.XmlElement) {
        const nodeName = node.nodeName;
        const attrs = Object.fromEntries(node.getAttributes());

        // Build canonical JSON representation
        const canonical = {
          type: nodeName,
          attrs: Object.keys(attrs).length > 0 ? attrs : undefined,
          content: []
        };

        // Extract text content
        let plainText = '';

        // Process children
        const children = node.toArray();
        for (let child of children) {
          if (child instanceof Y.XmlText) {
            const text = child.toString();
            plainText += text;
            canonical.content.push({ type: 'text', text });
          } else if (child instanceof Y.XmlElement) {
            // Recursively process child elements
            const childResult = extractFromNode(child, position);
            position = childResult;
          }
        }

        // Determine block type and heading level
        let blockType = nodeName;
        let level = null;

        if (nodeName === 'heading' || /^h[1-6]$/.test(nodeName)) {
          blockType = 'heading';
          level = attrs.level || parseInt(nodeName.charAt(1)) || 1;

          // Update headings path
          while (headingsPath.length >= level) {
            headingsPath.pop();
          }
          headingsPath[level - 1] = plainText;
        }

        // Only add block if it has content
        if (plainText.trim()) {
          blocks.push({
            type: blockType,
            level: level,
            headings_path: headingsPath.filter(Boolean).join(' > '),
            plain_text: plainText,
            canonical_json: JSON.stringify(canonical),
            position: position++
          });
        }

        return position;
      } else if (node instanceof Y.XmlText) {
        const text = node.toString();
        if (text.trim()) {
          blocks.push({
            type: 'text',
            level: null,
            headings_path: headingsPath.filter(Boolean).join(' > '),
            plain_text: text,
            canonical_json: JSON.stringify({ type: 'text', text }),
            position: position++
          });
        }
        return position;
      }

      return position;
    }

    // Extract all blocks from the document
    const children = sharedType.toArray();
    let position = 0;
    for (let child of children) {
      position = extractFromNode(child, position);
    }

    res.json({ blocks });
  } catch (err) {
    console.error('Error extracting blocks:', err);
    res.status(500).json({
      error: 'Failed to extract blocks',
      details: err.message
    });
  }
});

// Start server
app.listen(PORT, () => {
  console.log(`Yjs processor service listening on port ${PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully');
  process.exit(0);
});
