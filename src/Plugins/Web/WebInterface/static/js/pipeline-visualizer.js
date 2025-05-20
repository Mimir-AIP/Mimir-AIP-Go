/**
 * Pipeline Visualizer Component
 * Renders pipeline steps as a tree with status indicators
 */
class PipelineVisualizer {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.contentContainer = document.getElementById('pipeline-content');
        this.icons = {
            'completed': '✓',
            'running': '→',
            'pending': ' ',
            'failed': '✗'
        };
        this.currentTree = null;
        this.currentHighlightPath = null;
    }

    render(tree, highlightPath = null) {
        this.currentTree = tree;
        this.currentHighlightPath = highlightPath;
        this.contentContainer.innerHTML = this._renderNode(tree, '', true, highlightPath);
    }

    updateNodeStatus(path, status) {
        if (!this.currentTree) return;
        
        // Find the node in the tree
        let node = this.currentTree;
        for (const index of path) {
            if (!node.children || index >= node.children.length) return;
            node = node.children[index];
        }
        
        // Update status and re-render
        node.status = status;
        this.render(this.currentTree, this.currentHighlightPath);
    }

    _renderNode(node, prefix = '', isLast = true, highlightPath = null, currentPath = []) {
        if (!node) return '';
        // Status icon
        const status = node.status || 'pending';
        const icon = this.icons[status] || '[?]';
        
        // Highlight running node
        const highlight = highlightPath && this._pathsEqual(highlightPath, currentPath);
        const name = node.name || 'unnamed';
        
        let line = `${prefix}${icon} ${name}`;
        if (highlight) {
            line += " <== running";
        }
        
        // Children (substeps or iterations)
        let childrenHtml = '';
        const children = node.children || [];
        if (children.length > 0) {
            for (let i = 0; i < children.length; i++) {
                const child = children[i];
                const isChildLast = (i === children.length - 1);
                const branch = isLast ? '    ' : '│   ';
                const connector = isChildLast ? '└── ' : '├── ';
                childrenHtml += this._renderNode(
                    child, 
                    prefix + branch + connector, 
                    isChildLast, 
                    highlightPath, 
                    [...currentPath, i]
                );
            }
        }
        
        return `<div class="pipeline-node${highlight ? ' highlight' : ''}" data-path="${JSON.stringify(currentPath)}">
            <div class="pipeline-line">${line}</div>
            ${childrenHtml}
        </div>`;
    }

    _pathsEqual(path1, path2) {
        if (!path1 || !path2) return false;
        if (path1.length !== path2.length) return false;
        return path1.every((val, i) => val === path2[i]);
    }
}

// Export for Node/CommonJS
if (typeof module !== 'undefined' && module.exports) {
    module.exports = PipelineVisualizer;
}

// Global handler for pipeline visualization updates
window.updatePipeline = function(tree, highlightPath) {
    const visualizer = window.currentPipelineVisualizer;
    if (visualizer) {
        visualizer.render(tree, highlightPath);
    }
};