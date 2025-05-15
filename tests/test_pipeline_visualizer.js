import { PipelineVisualizer } from '../src/Plugins/WebInterface/static/js/pipeline-visualizer.js';

describe('PipelineVisualizer', () => {
  let visualizer;
  
  beforeEach(() => {
    visualizer = new PipelineVisualizer();
  });

  test('should initialize with empty state', () => {
    expect(visualizer.currentState).toEqual({});
  });

  test('should update visualization', () => {
    const testState = { nodes: [{ id: 1, status: 'running' }] };
    visualizer.updateVisualization(testState);
    expect(visualizer.currentState).toEqual(testState);
  });
});