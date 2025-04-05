# src/Plugins/PluginManager.py
import sys
import os
import importlib

class Plugin:
    def __init__(self, name, type):
        self.name = name
        self.type = type

class PluginManager:
    def __init__(self, plugins_path="Plugins"):
        self.plugins_path = plugins_path
        self.plugins = {}

        self._load_plugins()

    def _load_plugins(self):
        """Dynamically load all plugins in the Plugins folder."""
        plugin_base_path = os.path.join("src", self.plugins_path)
        plugin_types = os.listdir(plugin_base_path)

        for plugin_type in plugin_types:
            plugin_type_path = os.path.join(plugin_base_path, plugin_type)
            if os.path.isdir(plugin_type_path):
                for folder in os.listdir(plugin_type_path):
                    plugin_file = os.path.join(plugin_type_path, folder, f"{folder}.py")
                    if os.path.exists(plugin_file):
                        # Dynamically import the plugin
                        module_name = f"{plugin_type}.{os.path.basename(plugin_file)[:-3]}"
                        module = importlib.import_module(module_name)
                        # Assumes the class name matches the folder name
                        class_name = folder
                        plugin_class = getattr(module, class_name)
                        if hasattr(plugin_class, 'plugin_type') and plugin_class.plugin_type == plugin_type:
                            self.plugins.setdefault(plugin_type, {})[class_name] = plugin_class()

    def get_plugin(self, plugin_type, name: str):
        """Return a specific plugin by type and name."""
        return self.plugins.get(plugin_type, {}).get(name)

    def get_plugins(self, plugin_type=None):
        """Return all plugins of a specific type or all plugins if no type is specified."""
        if plugin_type:
            return self.plugins.get(plugin_type, {})
        return self.plugins

if __name__ == "__main__":
    manager = PluginManager()
    print(manager.get_plugins())