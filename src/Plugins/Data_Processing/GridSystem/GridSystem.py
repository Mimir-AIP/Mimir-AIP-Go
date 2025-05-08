"""
GridSystem module.

A generic grid-based spatial management system for handling 2D grid operations.
Supports pathfinding, distance calculations, and spatial queries.
"""

import numpy as np
from typing import List, Tuple, Dict, Any, Optional
from Plugins.BasePlugin import BasePlugin


class GridSystem(BasePlugin):
    """
    Generic grid system plugin for managing spatial relationships and operations.
    Provides functionality for:
    - Grid initialization and management
    - Cell occupancy tracking
    - Pathfinding
    - Range-based queries
    - Distance calculations
    """

    plugin_type = "Data_Processing"

    def __init__(self):
        """Initialize the GridSystem plugin"""
        self.grid = None
        self.cell_data = {}  # Store additional cell metadata
        self.default_grid_size = (10, 10)

    def initialize_grid(self, size: Tuple[int, int] = None) -> None:
        """Initialize or reset the grid with given dimensions
        
        Args:
            size: Tuple of (width, height) for the grid
        """
        if size is None:
            size = self.default_grid_size
        self.grid = np.zeros(size, dtype=int)
        self.cell_data = {}

    def set_cell(self, pos: Tuple[int, int], value: int = 1, metadata: Dict = None) -> None:
        """Set a cell's value and optional metadata
        
        Args:
            pos: (x, y) position in grid
            value: Integer value for the cell
            metadata: Optional dictionary of additional cell data
        """
        if self._is_valid_position(pos):
            self.grid[pos[1]][pos[0]] = value
            if metadata:
                self.cell_data[pos] = metadata

    def get_cell(self, pos: Tuple[int, int]) -> Dict[str, Any]:
        """Get cell value and metadata
        
        Args:
            pos: (x, y) position in grid
            
        Returns:
            Dictionary with cell value and metadata
        """
        if self._is_valid_position(pos):
            return {
                "value": self.grid[pos[1]][pos[0]],
                "metadata": self.cell_data.get(pos, {})
            }
        return {"value": None, "metadata": {}}

    def find_path(self, start: Tuple[int, int], end: Tuple[int, int]) -> List[Tuple[int, int]]:
        """Find path between two points using A* algorithm
        
        Args:
            start: Starting (x, y) position
            end: Target (x, y) position
            
        Returns:
            List of (x, y) positions forming the path
        """
        if not (self._is_valid_position(start) and self._is_valid_position(end)):
            return []

        # A* implementation
        open_set = {start}
        closed_set = set()
        came_from = {}
        g_score = {start: 0}
        f_score = {start: self._manhattan_distance(start, end)}

        while open_set:
            current = min(open_set, key=lambda pos: f_score.get(pos, float('inf')))
            if current == end:
                return self._reconstruct_path(came_from, current)

            open_set.remove(current)
            closed_set.add(current)

            for neighbor in self._get_neighbors(current):
                if neighbor in closed_set or self.grid[neighbor[1]][neighbor[0]] != 0:
                    continue

                tentative_g = g_score[current] + 1

                if neighbor not in open_set:
                    open_set.add(neighbor)
                elif tentative_g >= g_score.get(neighbor, float('inf')):
                    continue

                came_from[neighbor] = current
                g_score[neighbor] = tentative_g
                f_score[neighbor] = g_score[neighbor] + self._manhattan_distance(neighbor, end)

        return []

    def get_in_range(self, center: Tuple[int, int], range_val: int) -> List[Tuple[int, int]]:
        """Get all cells within given range of center point
        
        Args:
            center: (x, y) center position
            range_val: Range to search within
            
        Returns:
            List of (x, y) positions within range
        """
        if not self._is_valid_position(center):
            return []

        in_range = []
        for y in range(max(0, center[1] - range_val), min(self.grid.shape[0], center[1] + range_val + 1)):
            for x in range(max(0, center[0] - range_val), min(self.grid.shape[1], center[0] + range_val + 1)):
                if self._manhattan_distance((x, y), center) <= range_val:
                    in_range.append((x, y))
        return in_range

    def _is_valid_position(self, pos: Tuple[int, int]) -> bool:
        """Check if position is within grid bounds"""
        if self.grid is None:
            return False
        return (0 <= pos[0] < self.grid.shape[1] and 
                0 <= pos[1] < self.grid.shape[0])

    def _manhattan_distance(self, pos1: Tuple[int, int], pos2: Tuple[int, int]) -> int:
        """Calculate Manhattan distance between two points"""
        return abs(pos1[0] - pos2[0]) + abs(pos1[1] - pos2[1])

    def _get_neighbors(self, pos: Tuple[int, int]) -> List[Tuple[int, int]]:
        """Get valid neighboring cells"""
        neighbors = []
        for dx, dy in [(0, 1), (1, 0), (0, -1), (-1, 0)]:
            neighbor = (pos[0] + dx, pos[1] + dy)
            if self._is_valid_position(neighbor):
                neighbors.append(neighbor)
        return neighbors

    def _reconstruct_path(self, came_from: Dict[Tuple[int, int], Tuple[int, int]], 
                         current: Tuple[int, int]) -> List[Tuple[int, int]]:
        """Reconstruct path from came_from dictionary"""
        path = [current]
        while current in came_from:
            current = came_from[current]
            path.append(current)
        return path[::-1]

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step for this plugin
        
        Args:
            step_config: Configuration dictionary containing:
                - operation: Type of operation to perform
                    - 'init': Initialize grid
                    - 'set': Set cell values
                    - 'get': Get cell values
                    - 'path': Find path between points
                    - 'range': Get cells in range
                - grid_size: (width, height) for initialization
                - positions: List of positions for set/get operations
                - start/end: Positions for pathfinding
                - center/range: Position and range for range queries
            context: Pipeline context
            
        Returns:
            Dictionary containing operation results
        """
        config = step_config.get("config", {})
        operation = config.get("operation", "init")
        
        try:
            if operation == "init":
                size = tuple(config.get("grid_size", self.default_grid_size))
                self.initialize_grid(size)
                return {step_config["output"]: {"size": size}}
                
            elif operation == "set":
                positions = config.get("positions", [])
                for pos_data in positions:
                    pos = tuple(pos_data["position"])
                    value = pos_data.get("value", 1)
                    metadata = pos_data.get("metadata", {})
                    self.set_cell(pos, value, metadata)
                return {step_config["output"]: {"updated": len(positions)}}
                
            elif operation == "get":
                positions = config.get("positions", [])
                results = {tuple(pos): self.get_cell(tuple(pos)) for pos in positions}
                return {step_config["output"]: results}
                
            elif operation == "path":
                start = tuple(config["start"])
                end = tuple(config["end"])
                path = self.find_path(start, end)
                return {step_config["output"]: {"path": path, "length": len(path)}}
                
            elif operation == "range":
                center = tuple(config["center"])
                range_val = config["range"]
                cells = self.get_in_range(center, range_val)
                return {step_config["output"]: {"cells": cells, "count": len(cells)}}
                
            else:
                raise ValueError(f"Unknown operation: {operation}")
                
        except Exception as e:
            raise ValueError(f"Error in GridSystem plugin: {str(e)}")