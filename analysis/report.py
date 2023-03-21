import toml
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


def connection_string(config=toml.load("./db.toml")['psql']) -> str:
    return f"postgresql://{config['user']}:{config['password']}@{config['host']}:{config['port']}/{config['database']}"


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
        ORDER BY r.created_at
    """
    return pd.read_sql_query(query, con=conn)


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
        ORDER BY p.created_at
    """
    return pd.read_sql_query(query, con=conn)


def week_boxplots(data: pd.DataFrame, boxcolor: str, ylabel: str, title: str) -> plt.Figure:
    print(f"Plotting '{title}' boxplot graph...")

    regions = list(sorted(data["region"].unique()))

    fig, ax = plt.subplots(len(regions) // 2 + len(regions) % 2, 2, figsize=[14, 13], dpi=DPI)

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

    fig, ax = plt.subplots(1, 2, figsize=[15, 5], dpi=DPI)

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

    return fig


def regional_cdfs(retrievals: pd.DataFrame, publications: pd.DataFrame) -> plt.Figure:
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


def main():
    conn = sa.create_engine(connection_string())
    date_min = "2023-03-13"
    date_max = "2023-03-20"

    retrievals = get_retrievals(conn, date_min, date_max)
    publications = get_publications(conn, date_min, date_max)

    fig = week_boxplots(retrievals, "b", "Time to First Provider Record in s", "Retrievals")
    fig.savefig("./plots/retrievals_boxplot.png")

    fig = week_boxplots(publications, "r", "Publication Duration in s", "Publications")
    fig.savefig("./plots/publications_boxplot.png")

    fig = regional_boxplots(retrievals, publications)
    fig.savefig("./plots/regional_boxplot.png")

    fig = regional_cdfs(retrievals, publications)
    fig.savefig("./plots/regional_cdfs.png")


if __name__ == "__main__":
    main()
