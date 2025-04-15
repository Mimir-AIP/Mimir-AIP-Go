import importlib
import os
import sys
import inspect
from abc import ABC
from typing import Dict, Set

class PluginManager:
    def __init__(self, plugins_path="Plugins"):
        self.plugins_path = plugins_path
        self.plugins: Dict[str, Dict[str, object]] = {}
        self.warnings: Set[str] = set()
        
        # Add src directory to Python path for imports
        src_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        if src_dir not in sys.path:
            sys.path.append(src_dir)
            
        # Load plugins in the correct order
        self._load_plugins_by_type()
        
        # Print warnings
        for warning in sorted(self.warnings):
            print(warning)

    def _load_plugins_by_type(self):
        """Load plugins in a specific order to handle dependencies"""
        # First load AI Models as they are dependencies
        self._load_plugins_of_type("AIModels")
        
        # Then load everything else
        plugin_base_path = os.path.join("src", self.plugins_path)
        for plugin_type in os.listdir(plugin_base_path):
            if plugin_type != "AIModels" and os.path.isdir(os.path.join(plugin_base_path, plugin_type)):
                self._load_plugins_of_type(plugin_type)

    def _load_plugins_of_type(self, plugin_type: str):
        """Load all plugins of a specific type"""
        plugin_type_path = os.path.join("src", self.plugins_path, plugin_type)
        if not os.path.isdir(plugin_type_path):
            return

        for folder in os.listdir(plugin_type_path):
            if folder.startswith('__'):  # Skip __pycache__ and similar
                continue
                    
            plugin_file = os.path.join(plugin_type_path, folder, f"{folder}.py")
            if os.path.exists(plugin_file):
                try:
                    # Load module using importlib
                    module_name = f"Plugins.{plugin_type}.{folder}.{folder}"
                    
                    try:
                        module = importlib.import_module(module_name)

                        # Try different class name formats
                        base_name = folder.replace('-', '_')
                        class_names = [
                            base_name,  # snake_case
                            ''.join(word.capitalize() for word in base_name.split('_')),  # PascalCase
                            ''.join(word.capitalize() for word in base_name.split('_')) + 'Plugin',  # PascalCasePlugin
                        ]

                        plugin_class = None
                        for class_name in class_names:
                            if hasattr(module, class_name):
                                attr = getattr(module, class_name)
                                if hasattr(attr, 'plugin_type') and attr.plugin_type == plugin_type:
                                    # Skip abstract base classes
                                    if inspect.isclass(attr) and not inspect.isabstract(attr):
                                        plugin_class = attr
                                        break

                        if plugin_class:
                            try:
                                # Handle plugins that need dependencies
                                if plugin_type == "Data_Processing" and folder == "LLMFunction":
                                    # Get the first available AI Model plugin
                                    ai_plugins = self.get_plugins("AIModels")
                                    if ai_plugins:
                                        first_ai_plugin = next(iter(ai_plugins.values()))
                                        plugin_instance = plugin_class(llm_plugin=first_ai_plugin)
                                    else:
                                        self.warnings.add(f"Error instantiating plugin {folder}: No AI Model plugins available")
                                        continue
                                else:
                                    plugin_instance = plugin_class()
                                
                                self.plugins.setdefault(plugin_type, {})[folder] = plugin_instance
                                print(f"Successfully loaded plugin: {folder}")  # Debug print
                            except Exception as e:
                                self.warnings.add(f"Error instantiating plugin {folder}: {str(e)}")
                        else:
                            self.warnings.add(f"Warning: No valid plugin class found in {module_name}")

                    except ImportError as e:
                        self.warnings.add(f"Error importing module {module_name}: {str(e)}")

                except Exception as e:
                    self.warnings.add(f"Error loading plugin {folder}: {str(e)}")

    def get_plugins(self, plugin_type=None):
        """
        Get all plugins or plugins of a specific type

        Args:
            plugin_type (str): Type of plugin to get. If None, returns all plugins

        Returns:
            dict: Dictionary of plugins, where keys are plugin names and values are plugin instances
        """
        if plugin_type:
            return self.plugins.get(plugin_type, {})
        return self.plugins

    def get_plugin(self, plugin_type, name):
        """
        Get a specific plugin instance

        Args:
            plugin_type (str): Type of plugin (e.g., 'Input', 'Output', 'Data_Processing')
            name (str): Name of the plugin (e.g., 'rss_feed', 'HTMLReport')

        Returns:
            object: Plugin instance if found, None otherwise
        """
        if plugin_type not in self.plugins:
            return None
        
        if name not in self.plugins[plugin_type]:
            return None
        
        return self.plugins[plugin_type][name]

    def get_all_plugins(self):
        """Get all loaded plugins"""
        return self.get_plugins()

if __name__ == "__main__":
    manager = PluginManager()
    print(manager.get_plugins())