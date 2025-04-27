import logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler("mimir.log", mode="w"),
        logging.StreamHandler()
    ],
    force=True
)
logging.info("This should go to both file and console.")
