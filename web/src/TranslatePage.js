// Copyright 2021 The Casdoor Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import React from "react";
import {Link} from "react-router-dom";
import {Button, Popconfirm, Switch, Table} from 'antd';
import moment from "moment";
import * as Setting from "./Setting";
import * as OrganizationBackend from "./backend/OrganizationBackend";
import i18next from "i18next";

class TranslatePage extends React.Component {

  render() {
    const clearText = () => {
      document.getElementById("inputText").value = "";
    }

    return (
      <div>
        <div className="all">
          <div>
            <div className="model">
              <label htmlFor="model">Model: </label>
              <select id="model">
                <option value="batua_es2eu">Spanish -&gt; Basque</option>
                <option value="batua_eu2es">Basque -&gt; Spanish</option>
                <option value="generic_eu2en">Basque -&gt; English</option>
                <option value="generic_en2eu">English -&gt; Basque</option>
                <option value="generic_eu2fr">Basque -&gt; French</option>
                <option value="generic_fr2eu">French -&gt; Basque</option>
                <option value="generic_es2ca">Spanish -&gt; Catalan</option>
                <option value="generic_ca2es">Catalan -&gt; Spanish</option>
                <option value="generic_es2gl">Spanish -&gt; Galician</option>
                <option value="generic_gl2es">Galician -&gt; Spanish</option>
              </select>
            </div>
          </div>
          <div>
              <div className="toLeft">
                  <textarea rows="12" id="inputText" placeholder="Enter input text. It will be translated automatically."></textarea>
              </div>
              <div className="toRight">
                  <textarea rows="12"></textarea>
              </div>
          </div>
          <div className="all">
            <Button className="testingStyle" onClick={clearText}>Clear</Button>
          </div>
        </div>
      </div>
    );
  }
}

export default TranslatePage;
