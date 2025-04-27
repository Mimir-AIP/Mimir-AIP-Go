'''PluginManager module.

Discovers and loads plugins from the specified 'Plugins' directory. Each plugin must inherit from BasePlugin
and implement execute_pipeline_step(step_config, context).
'''

import importlib
import os
import sys
import inspect
from abc import ABC
from typing import Dict, Set
import logging

logging.basicConfig(level=logging.INFO)
logging.getLogger(__name__).info(f"[PluginManager:Startup] CWD: {os.getcwd()}, Python exec: {sys.executable}")

class PluginManager:
    '''Manages the discovery and loading of plugins from the specified 'Plugins' directory.

    Attributes:
        plugins_path (str): Path to the plugins directory.
        plugins (Dict[str, Dict[str, object]]): Dictionary of loaded plugins, where keys are plugin types and values are dictionaries of plugin instances.
        warnings (Set[str]): Set of warnings encountered during plugin loading.
    '''

    def __init__(self, plugins_path="Plugins"):
        '''Initializes the PluginManager instance.

        Args:
            plugins_path (str): Path to the plugins directory. Defaults to "Plugins".
        '''
        self.plugins_path = plugins_path
        self.plugins: Dict[str, Dict[str, object]] = {}
        self.warnings: Set[str] = set()
        
        # Add src directory to Python path for imports
        src_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        if src_dir not in sys.path:
            sys.path.append(src_dir)
            
        # Load plugins in the correct order
        self._load_plugins_by_type()
        
        # Print loaded plugin summary at info level
        loaded_plugins = {ptype: list(pdict.keys()) for ptype, pdict in self.plugins.items() if pdict}
        logging.getLogger(__name__).info(f"Loaded plugins: {loaded_plugins}")
        
        # Print loaded AIModel plugins after initial load (optional)
        ai_models = self.get_plugins("AIModels")
        if ai_models:
            logging.getLogger(__name__).info(f"Loaded AIModel plugins: {list(ai_models.keys())}")
        else:
            logging.getLogger(__name__).warning("No AIModel plugins loaded!")
        # Print warnings
        for warning in sorted(self.warnings):
            logging.getLogger(__name__).warning(warning)

    def _load_plugins_by_type(self):
        '''Loads plugins in a specific order to handle dependencies.'''
        # First load AI Models as they are dependencies
        self._load_plugins_of_type("AIModels")
        
        # Then load everything else
        # Use absolute path relative to project root, not current working directory
        project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        plugin_base_path = os.path.join(project_root, self.plugins_path)
        for plugin_type in os.listdir(plugin_base_path):
            if plugin_type != "AIModels" and os.path.isdir(os.path.join(plugin_base_path, plugin_type)):
                self._load_plugins_of_type(plugin_type)

    def _load_plugins_of_type(self, plugin_type: str):
        '''Loads all plugins of a specific type.

        Args:
            plugin_type (str): Type of plugin to load.
        '''
        project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        plugin_type_path = os.path.join(project_root, self.plugins_path, plugin_type)
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
                    logging.getLogger(__name__).info(f"[PluginManager] Attempting to import {module_name} from {plugin_file}")
                    try:
                        module = importlib.import_module(module_name)
                        logging.getLogger(__name__).info(f"[PluginManager] Imported module {module_name} from {module.__file__}")
                        # Debug: print out module namespace for diagnosis
                        logging.getLogger(__name__).debug(f"dir({module_name}): {dir(module)}")
                        # Try different class name formats
                        base_name = folder.replace('-', '_')
                        class_names = [
                            base_name,  # snake_case
                            ''.join(word.capitalize() for word in base_name.split('_')),  # PascalCase
                            ''.join(word.capitalize() for word in base_name.split('_')) + 'Plugin',  # PascalCasePlugin
                            folder,  # Original folder name (preserves case)
                            folder + 'Plugin',  # Original folder name + 'Plugin'
                        ]

                        plugin_class = None
                        for class_name in class_names:
                            if hasattr(module, class_name):
                                attr = getattr(module, class_name)
                                # Debug: log plugin_type and abstract status
                                plugin_type_val = getattr(attr, 'plugin_type', None)
                                is_abstract = inspect.isclass(attr) and inspect.isabstract(attr)
                                logging.getLogger(__name__).debug(
                                    f"Checking class '{class_name}': plugin_type={plugin_type_val}, is_abstract={is_abstract}"
                                )
                                # Skip abstract base classes (ABC) and classes with abstract methods
                                if inspect.isclass(attr) and not inspect.isabstract(attr):
                                    if hasattr(attr, 'plugin_type') and attr.plugin_type == plugin_type:
                                        plugin_class = attr
                                        break
                        if not plugin_class:
                            logging.getLogger(__name__).warning(f"No plugin class found in {module_name} for names {class_names}")

                        if plugin_class:
                            try:
                                # Handle plugins that need dependencies
                                if plugin_type == "Data_Processing" and folder == "LLMFunction":
                                    # Get the first available AI Model plugin and pass plugin_manager for dynamic selection
                                    ai_plugins = self.get_plugins("AIModels")
                                    if ai_plugins:
                                        first_ai_plugin = next(iter(ai_plugins.values()))
                                        plugin_instance = plugin_class(llm_plugin=first_ai_plugin, plugin_manager=self)
                                    else:
                                        self.warnings.add(f"Error instantiating plugin {folder}: No AI Model plugins available")
                                        continue
                                else:
                                    plugin_instance = plugin_class()
                                logging.getLogger(__name__).info(f"[PluginManager] Instantiated {plugin_class} from {plugin_file}")
                                self.plugins.setdefault(plugin_type, {})[folder] = plugin_instance
                                logging.getLogger(__name__).info(f"Successfully loaded plugin: {folder}")  
                            except Exception as e:
                                self.warnings.add(f"Error instantiating plugin {folder}: {str(e)}")

                    except ImportError as e:
                        self.warnings.add(f"Error importing module {module_name}: {str(e)}")

                except Exception as e:
                    self.warnings.add(f"Error loading plugin {folder}: {str(e)}")

    def get_plugins(self, plugin_type=None):
        '''Gets all plugins or plugins of a specific type.

        Args:
            plugin_type (str): Type of plugin to get. If None, returns all plugins.

        Returns:
            dict: Dictionary of plugins, where keys are plugin names and values are plugin instances.
        '''
        if plugin_type:
            return self.plugins.get(plugin_type, {})
        return self.plugins

    def get_plugin(self, plugin_type, name):
        '''Gets a specific plugin instance.

        Args:
            plugin_type (str): Type of plugin (e.g., 'Input', 'Output', 'Data_Processing')
            name (str): Name of the plugin (e.g., 'rss_feed', 'HTMLReport')

        Returns:
            object: Plugin instance if found, None otherwise.
        '''
        if plugin_type not in self.plugins:
            return None
        
        if name not in self.plugins[plugin_type]:
            return None
        
        return self.plugins[plugin_type][name]

    def get_all_plugins(self):
        '''Gets all loaded plugins.'''
        return self.get_plugins()

if __name__ == "__main__":
    manager = PluginManager()
    logging.getLogger(__name__).info(manager.get_plugins())