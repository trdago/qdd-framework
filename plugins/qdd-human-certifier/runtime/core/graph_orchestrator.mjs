/**
 * QDD Graph Orchestrator - DAG Dependency Resolver
 * Strictly follows Zero-Else and Early Return.
 */

export class GraphOrchestrator {
  constructor() {
    this.nodes = new Map();
  }

  registerNode(node) {
    if (!node || !node.id) {
      throw new Error('Invalid node definition: missing node.id');
    }
    this.nodes.set(node.id, {
      ...node,
      dependencies: Array.isArray(node.dependencies) ? node.dependencies : []
    });
  }

  registerNodes(nodeList) {
    if (!Array.isArray(nodeList)) {
      return;
    }
    for (const node of nodeList) {
      this.registerNode(node);
    }
  }

  resolveExecutionPlan(targetPattern = 'all') {
    const selectedNodes = this.filterTargetNodes(targetPattern);
    const requiredNodeIds = new Set();

    for (const node of selectedNodes) {
      this.collectDependencies(node.id, requiredNodeIds, new Set());
    }

    return this.topologicalSort(Array.from(requiredNodeIds));
  }

  filterTargetNodes(targetPattern) {
    if (!targetPattern || targetPattern === 'all') {
      return Array.from(this.nodes.values());
    }

    if (this.nodes.has(targetPattern)) {
      return [this.nodes.get(targetPattern)];
    }

    // Match by category
    const categoryMatches = Array.from(this.nodes.values()).filter(
      (n) => n.category === targetPattern
    );
    if (categoryMatches.length > 0) {
      return categoryMatches;
    }

    // Match by prefix
    const prefixMatches = Array.from(this.nodes.values()).filter((n) =>
      n.id.startsWith(targetPattern)
    );
    if (prefixMatches.length > 0) {
      return prefixMatches;
    }

    throw new Error(`Target '${targetPattern}' does not match any registered unit or category.`);
  }

  collectDependencies(nodeId, collectedSet, visitedStack) {
    if (visitedStack.has(nodeId)) {
      throw new Error(`Circular dependency detected at node '${nodeId}'`);
    }
    if (collectedSet.has(nodeId)) {
      return;
    }

    const node = this.nodes.get(nodeId);
    if (!node) {
      throw new Error(`Unresolved dependency '${nodeId}'`);
    }

    visitedStack.add(nodeId);
    for (const depId of node.dependencies) {
      this.collectDependencies(depId, collectedSet, visitedStack);
    }
    visitedStack.delete(nodeId);
    collectedSet.add(nodeId);
  }

  topologicalSort(nodeIds) {
    const inDegree = new Map();
    const adjList = new Map();

    for (const id of nodeIds) {
      inDegree.set(id, 0);
      adjList.set(id, []);
    }

    for (const id of nodeIds) {
      const node = this.nodes.get(id);
      for (const depId of node.dependencies) {
        if (!inDegree.has(depId)) {
          continue;
        }
        adjList.get(depId).push(id);
        inDegree.set(id, inDegree.get(id) + 1);
      }
    }

    const queue = [];
    for (const [id, deg] of inDegree.entries()) {
      if (deg === 0) {
        queue.push(id);
      }
    }

    const sorted = [];
    while (queue.length > 0) {
      const currentId = queue.shift();
      sorted.push(this.nodes.get(currentId));

      for (const neighborId of adjList.get(currentId)) {
        const nextDeg = inDegree.get(neighborId) - 1;
        inDegree.set(neighborId, nextDeg);
        if (nextDeg === 0) {
          queue.push(neighborId);
        }
      }
    }

    if (sorted.length !== nodeIds.length) {
      throw new Error('Graph cycle detected during topological resolution');
    }

    return sorted;
  }
}
