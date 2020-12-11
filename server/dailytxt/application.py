from flask import Flask
from flask_cors import CORS


def create_app(app_name='DailyTxT'):
    app = Flask(app_name)
    app.config.from_object('dailytxt.config.BaseConfig')

    cors = CORS(app, resources={r"/api/*": {"origins": "*"}})

    from dailytxt.api import api
    app.register_blueprint(api, url_prefix="/api")

    return app
