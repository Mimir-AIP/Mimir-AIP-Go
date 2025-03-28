import os
import importlib

def load_plugins():
    plugins_dir = 'plugins'
    plugins = {}
    for category in os.listdir(plugins_dir):
        category_dir = os.path.join(plugins_dir, category)
        if os.path.isdir(category_dir):
            for plugin in os.listdir(category_dir):
                plugin_dir = os.path.join(category_dir, plugin)
                if os.path.isdir(plugin_dir):
                    plugin_module = importlib.import_module(f'{category}.{plugin}.{plugin}')
                    plugins[plugin] = plugin_module
    return plugins