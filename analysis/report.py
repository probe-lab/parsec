import toml
import os
import sys
import datetime as dt
import math
import pickle
import pandas as pd
import numpy as np
import seaborn as sns
import matplotlib.pyplot as plt
import matplotlib as mpl
import sqlalchemy as sa
import datetime
from matplotlib.patches import Patch
from matplotlib.lines import Line2D
from mpl_toolkits.axes_grid1.axes_divider import make_axes_locatable

sns.set_theme()

# Constants
DPI = 150


def connection_string(config) -> str:
    return f"postgresql://{config.get('user')}:{config.get('password')}@{config.get('host')}:{config.get('port')}/{config.get('database')}?sslmode={config.get('sslmode','prefer')}"


def cdf(series: pd.Series) -> pd.DataFrame:
    """ calculates the cumulative distribution function of the given series"""
    return pd.DataFrame.from_dict({
        series.name: np.append(series.sort_values(), series.max()),
        "cdf": np.linspace(0, 1, len(series) + 1)
    })


def get_retrievals(conn: sa.engine.Engine, start_date: str, end_date: str) -> pd.DataFrame:
    print("Loading retrievals...")
    query = f"""
        SELECT
            n.region,
            r.duration,
            r.created_at,
            DATE(r.created_at)::TEXT date,
            r.error IS NOT NULL has_error
        FROM retrievals r
            INNER JOIN nodes n ON r.node_id = n.id
        WHERE r.created_at >= '{start_date}'
          AND r.created_at < '{end_date}'
          AND r.rt_size > 200
          AND n.instance_type = 't3.small'
        ORDER BY r.created_at
    """
    legacy = pd.read_sql_query(query, con=conn)

    query = f"""
        SELECT
            n.region,
            r.duration,
            r.created_at,
            DATE(r.created_at)::TEXT date,
            r.error IS NOT NULL has_error
        FROM retrievals_ecs r
            INNER JOIN nodes_ecs n ON r.node_id = n.id
        WHERE r.created_at >= '{start_date}'
          AND r.created_at < '{end_date}'
          AND r.rt_size > 200
          AND n.fleet = 'default'
        ORDER BY r.created_at
    """
    ecs = pd.read_sql_query(query, con=conn)

    return pd.concat([legacy, ecs]).reset_index()

def get_publications(conn: sa.engine.Engine, start_date: str, end_date: str) -> pd.DataFrame:
    print("Loading publications...")
    query = f"""
        SELECT
            n.region,
            p.duration,
            p.created_at,
            DATE(p.created_at)::TEXT date,
            p.error IS NOT NULL has_error
        FROM provides p
            INNER JOIN nodes n ON p.node_id = n.id
        WHERE p.created_at >= '{start_date}'
          AND p.created_at < '{end_date}'
          AND p.rt_size > 190
          AND n.instance_type = 't3.small'
        ORDER BY p.created_at
    """
    legacy = pd.read_sql_query(query, con=conn)

    query = f"""
        SELECT
            n.region,
            p.duration,
            p.created_at,
            DATE(p.created_at)::TEXT date,
            p.error IS NOT NULL has_error
        FROM provides_ecs p
            INNER JOIN nodes_ecs n ON p.node_id = n.id
        WHERE p.created_at >= '{start_date}'
          AND p.created_at < '{end_date}'
          AND p.rt_size > 190
          AND n.fleet = 'default'
        ORDER BY p.created_at
    """
    ecs = pd.read_sql_query(query, con=conn)

    return pd.concat([legacy, ecs]).reset_index()


def week_boxplots(data: pd.DataFrame, boxcolor: str, ylabel: str, title: str) -> plt.Figure:
    if len(data) == 0:
        return plt.Figure()

    print(f"Plotting '{title}' boxplot graph...")
    regions = list(sorted(data["region"].unique()))

    fig, ax = plt.subplots(len(regions) // 2 + len(regions) % 2, 2, figsize=[14, 15], dpi=DPI)

    for i, region in enumerate(regions):
        ax = fig.axes[i]

        group = data[data["region"] == region].groupby('date')
        grouped = group['duration'].apply(list)

        bplot = ax.boxplot(grouped, notch=True, showfliers=False, labels=grouped.index, patch_artist=True)

        for box in bplot["boxes"]:
            box.set_facecolor(boxcolor)

        for median in bplot["medians"]:
            median.set_color("white")

        ax.set_xlabel("Date")
        ax.set_title(f"Region {region} ")
        ax.set_ylabel(ylabel)
        ax.set_ylim(0)

        for j, index in enumerate(grouped.index):
            y = np.percentile(grouped.loc[index], 60)
            ax.text(j + 1, y, format(len(grouped.loc[index]), ","), ha="center", fontdict={"fontsize": 10}, color="w")

        for tick in ax.get_xticklabels():
            tick.set_rotation(10)
            tick.set_ha("right")
    fig.suptitle(title)
    fig.tight_layout()

    return fig


def regional_boxplots(retrievals: pd.DataFrame, provides: pd.DataFrame) -> plt.Figure:
    if len(retrievals) == 0 or len(provides) == 0:
        return plt.Figure()

    plots = [
        {
            "data": provides,
            "boxcolor": "r",
            "ylabel": "Publication Duration in s",
            "title": "Publications",
        },
        {
            "data": retrievals,
            "boxcolor": "b",
            "ylabel": "Time to First Provider Record in s",
            "title": "Retrievals",
        },
    ]

    fig, ax = plt.subplots(1, 2, figsize=[13, 4], dpi=DPI)

    for i, plot in enumerate(plots):
        print(f"Plotting regional boxplot graph...")
        ax = fig.axes[i]

        group = plot['data'].groupby('region')
        data = group['duration'].apply(list)

        bplot = ax.boxplot(data, notch=True, showfliers=False, labels=data.index, patch_artist=True)

        for box in bplot["boxes"]:
            box.set_facecolor(plot["boxcolor"])

        for median in bplot["medians"]:
            median.set_color("white")

        ax.set_title(f"{plot['title']} from {plot['data']['date'].min()} to {plot['data']['date'].max()}")
        ax.set_xlabel("Region")
        ax.set_ylabel(plot['ylabel'])
        ax.set_ylim(0)

        for j, index in enumerate(data.index):
            y = np.percentile(data.loc[index], 60)
            ax.text(j + 1, y, format(len(data.loc[index]), ","), ha="center", fontdict={"fontsize": 10}, color="w")

        for tick in ax.get_xticklabels():
            tick.set_rotation(10)
            tick.set_ha("right")

    fig.tight_layout()

    return fig


def regional_cdfs(retrievals: pd.DataFrame, publications: pd.DataFrame) -> plt.Figure:
    if len(retrievals) == 0 or len(publications) == 0:
        return plt.Figure()

    plots = [
        {
            "data": publications,
            "ylabel": "Publication Duration in s",
            "title": "Publications",
            "ymax": 75,
        },
        {
            "data": retrievals,
            "ylabel": "Time to First Provider Record in s",
            "title": "Retrievals",
            "ymax": 2.5,
        },
    ]

    fig, ax = plt.subplots(1, 2, figsize=[15, 5], dpi=DPI)

    for i, plot in enumerate(plots):
        ax = fig.axes[i]

        group = plot["data"].groupby('region')
        data = group['duration'].apply(list)

        for j, region in enumerate(sorted(plot["data"]["region"].unique())):
            dat = cdf(pd.Series(data.loc[region], name="duration"))
            ax.plot(dat["duration"], dat["cdf"], label=f"{region} ({format(len(dat), ',')})")

        ax.set_xlim(-0.1, plot["ymax"])
        ax.legend(title="Region")
        ax.set_xlabel(plot["ylabel"])
        ax.set_ylabel("CDF")
        ax.set_title(f"{plot['title']} ({plot['data']['date'].min()} to {plot['data']['date'].max()})")

    fig.tight_layout()

    return fig


def errors(provides: pd.DataFrame, retrievals: pd.DataFrame) -> plt.Figure:
    if len(retrievals) == 0 or len(provides) == 0:
        return plt.Figure()

    plots = [
        {
            "data": provides,
            "color": "r",
            "type": "Provides",
        },
        {
            "data": retrievals,
            "color": "b",
            "type": "Retrievals",
        }
    ]

    fig, ax = plt.subplots(1, 2, figsize=[15, 5], dpi=150)
    for i, plot in enumerate(plots):
        data = plot["data"].groupby(["region"])["has_error"].agg(['mean', 'count']).reset_index()

        ax = fig.axes[i]
        ax.bar(data["region"], 100 * data["mean"], color=plot["color"])

        y = 0.05 * data['mean'].max()
        for i, region in enumerate(data["region"]):
            ax.text(i, 100 * (data['mean'][i] + y), f"Total {plot['type']}\n{format(int(data['count'][i]), ',')}",
                    ha="center", va="bottom", bbox={"facecolor": 'white'}, fontsize=8)

            ax.text(i, 100 * y, f"{int((data['mean'] * data['count'])[i])} Failed\n{plot['type']}",
                    ha="center", va="bottom", bbox={"facecolor": 'white'}, fontsize=8)

        ax.set_ylim(0, 100 * data['mean'].max() * 1.2)
        ax.set_xlabel("Region")
        ax.set_ylabel(f"Errors in % of Total {plot['type']}")

        for tick in ax.get_xticklabels():
            tick.set_rotation(10)
            tick.set_ha("right")

    fig.tight_layout()

    return fig

def main():
    if os.environ.get('PARSEC_DATABASE_HOST') is not None:
        db_config = {
            'host': os.environ['PARSEC_DATABASE_HOST'],
            'port': os.environ['PARSEC_DATABASE_PORT'],
            'database': os.environ['PARSEC_DATABASE_NAME'],
            'user': os.environ['PARSEC_DATABASE_USER'],
            'password': os.environ['PARSEC_DATABASE_PASSWORD'],
            'sslmode': os.environ['PARSEC_DATABASE_SSL_MODE'],
        }
    else:
        db_config=toml.load("./db.toml")['psql']

    conn = sa.create_engine(connection_string(db_config))

    now = dt.datetime.today()
    year = os.getenv('PARSEC_REPORT_YEAR', now.year)
    calendar_week = os.getenv('PARSEC_REPORT_WEEK', now.isocalendar().week - 1)
    date_min = dt.datetime.strptime(f"{year}-W{calendar_week}-1", "%Y-W%W-%w")
    date_max = date_min + dt.timedelta(weeks=1)

    if len(sys.argv) > 1:
        output_dir = sys.argv[1]
    else:
        output_dir = '.'

    # allow name of plots directory to be overridden
    plots_dirname = os.getenv('PARSEC_PLOTS_DIRNAME', 'plots')
    plots_dir = os.path.join(output_dir, plots_dirname)
    if not os.path.isdir(plots_dir):
        os.mkdir(plots_dir)

    print(f'Generating report for year {year}, week {calendar_week}')
    print(f'Date range from {date_min} to {date_max}')
    print(f'Writing plots to {plots_dir}')
    print('Using database connection with:')
    print('host: ' + db_config['host'])
    print('port: ' + str(db_config['port']))
    print('database: ' + db_config['database'])
    print('user: ' + db_config['user'])
    print('sslmode: ' + db_config.get('sslmode','prefer'))

    retrievals = get_retrievals(conn, date_min, date_max)
    publications = get_publications(conn, date_min, date_max)

    fig = week_boxplots(retrievals, "b", "Time to First Provider Record in s", "Retrievals")
    fig.savefig(f"{plots_dir}/parsec-retrievals-boxplot-daily.png")

    fig = week_boxplots(publications, "r", "Publication Duration in s", "Publications")
    fig.savefig(f"{plots_dir}/parsec-publications-boxplot-daily.png")

    fig = regional_boxplots(retrievals, publications)
    fig.savefig(f"{plots_dir}/parsec-regions-boxplot.png")

    fig = regional_cdfs(retrievals, publications)
    fig.savefig(f"{plots_dir}/parsec-regions-cdf.png")

    fig = errors(retrievals, publications)
    fig.savefig(f"{plots_dir}/parsec-error-rate.png")


if __name__ == "__main__":
    main()
