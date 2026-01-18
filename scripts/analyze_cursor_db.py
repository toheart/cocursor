#!/usr/bin/env python3
"""
Cursor 数据库分析脚本
分析全局存储和工作区存储的 Cursor 数据库
"""

import os
import sqlite3
import json
import urllib.parse
from pathlib import Path
from typing import Dict, List, Tuple, Any
from collections import defaultdict


class CursorDBAnalyzer:
    """Cursor 数据库分析器"""

    def __init__(self):
        # Cursor 数据路径
        self.cursor_data_path = Path(r"C:\Users\TANG\AppData\Roaming\Cursor\User")
        self.global_db_path = self.cursor_data_path / "globalStorage" / "state.vscdb"
        self.workspace_storage_path = self.cursor_data_path / "workspaceStorage"

        # 存储分析结果
        self.workspaces = {}  # workspace_id -> workspace_info
        self.global_data = {}  # key -> value

    def load_global_data(self):
        """加载全局存储数据"""
        print("=== 加载全局存储数据 ===")
        if not self.global_db_path.exists():
            print(f"错误：全局数据库不存在: {self.global_db_path}")
            return

        conn = sqlite3.connect(self.global_db_path)
        cursor = conn.cursor()

        # 查询所有数据
        cursor.execute("SELECT key, value FROM ItemTable")
        rows = cursor.fetchall()

        print(f"全局存储记录数: {len(rows)}")

        for key, value in rows:
            try:
                # 尝试解析为文本
                if value:
                    value_str = value.decode('utf-8')
                    self.global_data[key] = value_str
            except Exception as e:
                self.global_data[key] = f"<BLOB: {len(value)} bytes>"

        conn.close()
        print(f"成功加载 {len(self.global_data)} 条全局记录\n")

    def discover_workspaces(self):
        """发现所有工作区"""
        print("=== 发现工作区 ===")

        if not self.workspace_storage_path.exists():
            print(f"错误：工作区存储路径不存在: {self.workspace_storage_path}")
            return

        # 扫描所有工作区目录
        workspace_dirs = [d for d in self.workspace_storage_path.iterdir() if d.is_dir()]

        print(f"发现 {len(workspace_dirs)} 个工作区目录\n")

        for workspace_dir in workspace_dirs:
            workspace_id = workspace_dir.name

            # 读取 workspace.json
            workspace_json = workspace_dir / "workspace.json"
            if workspace_json.exists():
                with open(workspace_json, 'r', encoding='utf-8') as f:
                    ws_data = json.load(f)

                # 解码路径
                folder_uri = ws_data.get('folder', '')
                folder_path = urllib.parse.unquote(folder_uri.replace('file:///', ''))

                # 提取项目名（从路径最后一个目录名）
                project_name = Path(folder_path).name

                # 检查数据库文件
                db_path = workspace_dir / "state.vscdb"
                db_exists = db_path.exists()

                self.workspaces[workspace_id] = {
                    'workspace_id': workspace_id,
                    'folder_uri': folder_uri,
                    'folder_path': folder_path,
                    'project_name': project_name,
                    'db_path': str(db_path) if db_exists else None,
                    'db_exists': db_exists,
                }

                print(f"工作区: {workspace_id}")
                print(f"  路径: {folder_path}")
                print(f"  项目名: {project_name}")
                print(f"  数据库: {'存在' if db_exists else '不存在'}\n")

    def analyze_workspace_db(self, workspace_id: str):
        """分析单个工作区数据库"""
        workspace = self.workspaces.get(workspace_id)
        if not workspace or not workspace['db_exists']:
            return None

        db_path = workspace['db_path']
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()

        # 统计数据
        stats = {
            'total_records': 0,
            'ai_service_prompts': 0,
            'ai_service_generations': 0,
            'composer_data': 0,
            'total_sizes': {},
        }

        # 查询所有数据
        cursor.execute("SELECT key, value FROM ItemTable")
        rows = cursor.fetchall()

        for key, value in rows:
            stats['total_records'] += 1
            if value:
                stats['total_sizes'][key] = len(value)

            # 分类统计
            if key == 'aiService.prompts':
                stats['ai_service_prompts'] = len(value) if value else 0
            elif key == 'aiService.generations':
                stats['ai_service_generations'] = len(value) if value else 0
            elif key == 'composer.composerData':
                stats['composer_data'] = len(value) if value else 0

        conn.close()
        return stats

    def analyze_all_workspaces(self):
        """分析所有工作区数据库"""
        print("=== 分析所有工作区数据库 ===\n")

        for workspace_id in self.workspaces:
            stats = self.analyze_workspace_db(workspace_id)
            if stats:
                self.workspaces[workspace_id]['stats'] = stats

                workspace = self.workspaces[workspace_id]
                print(f"工作区: {workspace['project_name']} ({workspace_id})")
                print(f"  总记录数: {stats['total_records']}")
                print(f"  AI Prompts: {stats['ai_service_prompts']} bytes")
                print(f"  AI Generations: {stats['ai_service_generations']} bytes")
                print(f"  Composer Data: {stats['composer_data']} bytes")
                print()

    def analyze_global_tracking(self):
        """分析全局 AI 代码追踪数据"""
        print("=== 分析全局 AI 代码追踪 ===")

        tracking_data = {}
        for key in self.global_data:
            if key.startswith('aiCodeTracking.dailyStats'):
                try:
                    value = json.loads(self.global_data[key])
                    tracking_data[key] = value
                except:
                    pass

        print(f"发现 {len(tracking_data)} 条追踪记录:\n")

        for key in sorted(tracking_data.keys()):
            data = tracking_data[key]
            print(f"日期: {key.split('.')[-1]}")
            print(f"  Tab 建议: {data.get('tabSuggestedLines', 0)} 行")
            print(f"  Tab 接受: {data.get('tabAcceptedLines', 0)} 行")
            print(f"  Composer 建议: {data.get('composerSuggestedLines', 0)} 行")
            print(f"  Composer 接受: {data.get('composerAcceptedLines', 0)} 行")

            # 计算接受率
            tab_suggested = data.get('tabSuggestedLines', 0)
            tab_accepted = data.get('tabAcceptedLines', 0)
            composer_suggested = data.get('composerSuggestedLines', 0)
            composer_accepted = data.get('composerAcceptedLines', 0)

            if tab_suggested > 0:
                tab_rate = (tab_accepted / tab_suggested) * 100
                print(f"  Tab 接受率: {tab_rate:.2f}%")
            if composer_suggested > 0:
                composer_rate = (composer_accepted / composer_suggested) * 100
                print(f"  Composer 接受率: {composer_rate:.2f}%")
            print()

    def generate_project_mapping(self):
        """生成项目名映射"""
        print("=== 生成项目名映射 ===\n")

        project_map = defaultdict(list)
        name_conflicts = []

        # 按项目名分组
        for workspace_id, workspace in self.workspaces.items():
            project_name = workspace['project_name']
            project_map[project_name].append(workspace_id)

            # 检查重名
            if len(project_map[project_name]) > 1:
                if project_name not in [c[0] for c in name_conflicts]:
                    name_conflicts.append((project_name, project_map[project_name]))

        # 输出映射关系
        print("项目名 → 工作区 ID 映射:")
        for project_name, workspace_ids in sorted(project_map.items()):
            if len(workspace_ids) == 1:
                print(f"  {project_name} → {workspace_ids[0]}")
            else:
                print(f"  {project_name} → {workspace_ids} (重名!)")

        # 输出配置文件建议
        print("\n建议的项目配置 (projects.json):")
        print("{")
        print("  \"projects\": {")
        for project_name in sorted(project_map.keys()):
            for i, workspace_id in enumerate(project_map[project_name]):
                workspace = self.workspaces[workspace_id]

                # 如果有重名，添加序号
                if len(project_map[project_name]) > 1:
                    display_name = f"{project_name}-{i + 1}"
                else:
                    display_name = project_name

                print(f"    \"{display_name}\": {{")
                print(f"      \"name\": \"{display_name}\",")
                print(f"      \"workspace_id\": \"{workspace_id}\",")
                print(f"      \"path\": \"{workspace['folder_path']}\"")
                if i < len(project_map[project_name]) - 1 or project_name != sorted(project_map.keys())[-1]:
                    print(f"    }},")
                else:
                    print(f"    }}")
        print("  }")
        print("}")

    def get_recent_projects(self, limit=10):
        """获取最近打开的项目"""
        print(f"=== 最近打开的项目 (Top {limit}) ===\n")

        recent_key = "history.recentlyOpenedPathsList"
        if recent_key not in self.global_data:
            print("未找到最近打开的项目记录")
            return

        try:
            recent_data = json.loads(self.global_data[recent_key])
            entries = recent_data.get('entries', [])

            for i, entry in enumerate(entries[:limit]):
                folder_uri = entry.get('folderUri', '')
                folder_path = urllib.parse.unquote(folder_uri.replace('file:///', ''))
                project_name = Path(folder_path).name

                print(f"{i + 1}. {project_name}")
                print(f"   路径: {folder_path}")

                # 尝试匹配工作区
                for ws_id, ws in self.workspaces.items():
                    if ws['folder_uri'] == folder_uri:
                        print(f"   工作区 ID: {ws_id}")
                        break
                print()

        except Exception as e:
            print(f"解析失败: {e}")

    def run_full_analysis(self):
        """运行完整分析"""
        print("Cursor 数据库分析工具")
        print("=" * 60)
        print()

        # 1. 加载全局数据
        self.load_global_data()

        # 2. 发现工作区
        self.discover_workspaces()

        # 3. 分析所有工作区数据库
        self.analyze_all_workspaces()

        # 4. 分析全局 AI 代码追踪
        self.analyze_global_tracking()

        # 5. 生成项目映射
        self.generate_project_mapping()

        # 6. 获取最近打开的项目
        self.get_recent_projects()

        print("=" * 60)
        print("分析完成!")


def main():
    """主函数"""
    analyzer = CursorDBAnalyzer()
    analyzer.run_full_analysis()


if __name__ == "__main__":
    main()
