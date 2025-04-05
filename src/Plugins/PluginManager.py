# src/Plugins/PluginManager.py
import os
import importlib
from Plugins.AIModels.BaseAIModel import BaseAIModel

class PluginManager:
    def __init__(self, plugins_path="Plugins/AIModels"):
        self.plugins_path = plugins_path
        self.plugins = {}

        self._load_plugins()

    def _load_plugins(self):
        """Dynamically load all plugins in the AIModels folder."""
        plugin_base_path = os.path.join("src", self.plugins_path)
        model_folders = os.listdir(plugin_base_path)

        for folder in model_folders:
            plugin_file = os.path.join(plugin_base_path, folder, f"{folder}.py")
            if os.path.exists(plugin_file):
                # Dynamically import the plugin
                module_name = f"Plugins.AIModels.{folder}.{folder}"
                module = importlib.import_module(module_name)
                # Assumes the class name matches the folder name
                class_name = folder
                plugin_class = getattr(module, class_name)
                if issubclass(plugin_class, BaseAIModel):
                    self.plugins[class_name] = plugin_class()

    def get_plugin(self, name: str) -> BaseAIModel:
        """Return a specific plugin by name."""
        return self.plugins.get(name)

    def get_all_plugins(self):
        """Return all loaded plugins."""
        return self.plugins

if __name__ == "__main__":
    manager = PluginManager()
    print(manager.get_all_plugins())