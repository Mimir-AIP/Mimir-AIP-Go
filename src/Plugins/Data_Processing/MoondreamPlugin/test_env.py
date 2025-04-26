import os
import dotenv
dotenv.load_dotenv(dotenv_path="./.env", verbose=True)
print("MOONDREAM_API_KEY:", repr(os.getenv("MOONDREAM_API_KEY")))