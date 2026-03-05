import pathlib

BASE_DIR = pathlib.Path(__file__).parent
PACKAGE_DIR = BASE_DIR.parent.parent
PROJECT_DIR = PACKAGE_DIR.parent.parent
DATA_DIR = PROJECT_DIR / "data"
