import yaml
import os
from typing import Dict, List, Any

def parse_pipeline_steps(steps: List[Dict[str, Any]], indent: int = 0) -> List[str]:
    """Recursively parse pipeline steps and generate simple flowchart nodes and edges."""
    mermaid = []
    
    for step in steps:
        # Create node for current step
        node_id = step['name'].replace(' ', '_')
        mermaid.append(f"    {node_id}[{step['name']}]")
        
        # Handle nested steps
        if 'steps' in step:
            nested_mermaid = parse_pipeline_steps(step['steps'], indent + 1)
            mermaid.extend(nested_mermaid)
            
            # Add end node for nested block
            end_id = f"{node_id}_end"
            mermaid.append(f"    {end_id}[End]")
            mermaid.append(f"    {step['steps'][-1]['name'].replace(' ', '_')} --> {end_id}")
            mermaid.append(f"    {node_id} --> {end_id}")
        
        # Connect to next step if not last
        if steps.index(step) < len(steps) - 1:
            next_step_id = steps[steps.index(step) + 1]['name'].replace(' ', '_')
            mermaid.append(f"    {node_id} --> {next_step_id}")
    
    return mermaid

def generate_mermaid_chart(pipeline_path: str, output_path: str) -> None:
    """Generate a simple MermaidJS flowchart from a pipeline configuration file."""
    with open(pipeline_path, 'r') as f:
        pipeline_config = yaml.safe_load(f)
    
    # Create markdown file with MermaidJS chart
    with open(output_path, 'w') as f:
        f.write(f"# {pipeline_config['pipelines'][0]['name']}\n\n")
        f.write("```mermaid\n")
        f.write("graph TD\n")
        
        # Generate the flowchart
        steps = pipeline_config['pipelines'][0]['steps']
        mermaid_lines = parse_pipeline_steps(steps)
        f.write("\n".join(mermaid_lines))
        f.write("\n```")

if __name__ == "__main__":
    pipeline_path = "src/pipelines/POC.yaml"
    output_path = "src/pipelines/POC_flowchart.md"
    generate_mermaid_chart(pipeline_path, output_path)
    print(f"Flowchart generated at: {output_path}")