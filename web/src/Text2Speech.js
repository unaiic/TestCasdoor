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

class Text2Speech extends React.Component {

  render() {
    const clearText = () => {
      document.getElementById("inputText").value = "";
    }

    return (
      <div>
        <div>
          <p style={{fontWeight: 'bold'}}>Archivo con texto: </p>
          <input type="file"></input>
        </div>
        <div style={{marginTop: '1%'}}>
          <Button>Generar</Button>
        </div>
        <div style={{display: 'none'}}>
          <p>Se enviará el archivo al email</p>
        </div>
      </div>
    );
  }
}

export default Text2Speech;
