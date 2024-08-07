GET / HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Vary: Accept-Encoding
Content-Type: text/html


<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <!--
    Modified from the Debian original for Ubuntu
    Last updated: 2014-03-19
    See: https://launchpad.net/bugs/1288690
  -->
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <title>Apache2 Ubuntu Default Page: It works</title>
    <style type="text/css" media="screen">
  * {
    margin: 0px 0px 0px 0px;
    padding: 0px 0px 0px 0px;
  }

  body, html {
    padding: 3px 3px 3px 3px;

    background-color: #D8DBE2;

    font-family: Verdana, sans-serif;
    font-size: 11pt;
    text-align: center;
  }

  div.main_page {
    position: relative;
    display: table;

    width: 800px;

    margin-bottom: 3px;
    margin-left: auto;
    margin-right: auto;
    padding: 0px 0px 0px 0px;

    border-width: 2px;
    border-color: #212738;
    border-style: solid;

    background-color: #FFFFFF;

    text-align: center;
  }

  div.page_header {
    height: 99px;
    width: 100%;

    background-color: #F5F6F7;
  }

  div.page_header span {
    margin: 15px 0px 0px 50px;

    font-size: 180%;
    font-weight: bold;
  }

  div.page_header img {
    margin: 3px 0px 0px 40px;

    border: 0px 0px 0px;
  }

  div.table_of_contents {
    clear: left;

    min-width: 200px;

    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.table_of_contents_item {
    clear: left;

    width: 100%;

    margin: 4px 0px 0px 0px;

    background-color: #FFFFFF;

    color: #000000;
    text-align: left;
  }

  div.table_of_contents_item a {
    margin: 6px 0px 0px 6px;
  }

  div.content_section {
    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.content_section_text {
    padding: 4px 8px 4px 8px;

    color: #000000;
    font-size: 100%;
  }

  div.content_section_text pre {
    margin: 8px 0px 8px 0px;
    padding: 8px 8px 8px 8px;

    border-width: 1px;
    border-style: dotted;
    border-color: #000000;

    background-color: #F5F6F7;

    font-style: italic;
  }

  div.content_section_text p {
    margin-bottom: 6px;
  }

  div.content_section_text ul, div.content_section_text li {
    padding: 4px 8px 4px 16px;
  }

  div.section_header {
    padding: 3px 6px 3px 6px;

    background-color: #8E9CB2;

    color: #FFFFFF;
    font-weight: bold;
    font-size: 112%;
    text-align: center;
  }

  div.section_header_red {
    background-color: #CD214F;
  }

  div.section_header_grey {
    background-color: #9F9386;
  }

  .floating_element {
    position: relative;
    float: left;
  }

  div.table_of_contents_item a,
  div.content_section_text a {
    text-decoration: none;
    font-weight: bold;
  }

  div.table_of_contents_item a:link,
  div.table_of_contents_item a:visited,
  div.table_of_contents_item a:active {
    color: #000000;
  }

  div.table_of_contents_item a:hover {
    background-color: #000000;

    color: #FFFFFF;
  }

  div.content_section_text a:link,
  div.content_section_text a:visited,
   div.content_section_text a:active {
    background-color: #DCDFE6;

    color: #000000;
  }

  div.content_section_text a:hover {
    background-color: #000000;

    color: #DCDFE6;
  }

  div.validator {
  }
    </style>
  </head>
  <body>
    <div class="main_page">
      <div class="page_header floating_element">
        <img src="/icons/ubuntu-logo.png" alt="Ubuntu Logo" class="floating_element"/>
        <span class="floating_element">
          Apache2 Ubuntu Default Page
        </span>
      </div>
<!--      <div class="table_of_contents floating_element">
        <div class="section_header section_header_grey">
          TABLE OF CONTENTS
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#about">About</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#changes">Changes</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#scope">Scope</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#files">Config files</a>
        </div>
      </div>
-->
      <div class="content_section floating_element">


        <div class="section_header section_header_red">
          <div id="about"></div>
          It works!
        </div>
        <div class="content_section_text">
          <p>
                This is the default welcome page used to test the correct 
                operation of the Apache2 server after installation on Ubuntu systems.
                It is based on the equivalent page on Debian, from which the Ubuntu Apache
                packaging is derived.
                If you can read this page, it means that the Apache HTTP server installed at
                this site is working properly. You should <b>replace this file</b> (located at
                <tt>/var/www/html/index.html</tt>) before continuing to operate your HTTP server.
          </p>


          <p>
                If you are a normal user of this web site and don't know what this page is
                about, this probably means that the site is currently unavailable due to
                maintenance.
                If the problem persists, please contact the site's administrator.
          </p>

        </div>
        <div class="section_header">
          <div id="changes"></div>
                Configuration Overview
        </div>
        <div class="content_section_text">
          <p>
                Ubuntu's Apache2 default configuration is different from the
                upstream default configuration, and split into several files optimized for
                interaction with Ubuntu tools. The configuration system is
                <b>fully documented in
                /usr/share/doc/apache2/README.Debian.gz</b>. Refer to this for the full
                documentation. Documentation for the web server itself can be
                found by accessing the <a href="/manual">manual</a> if the <tt>apache2-doc</tt>
                package was installed on this server.

          </p>
          <p>
                The configuration layout for an Apache2 web server installation on Ubuntu systems is as follows:
          </p>
          <pre>
/etc/apache2/
|-- apache2.conf
|       `--  ports.conf
|-- mods-enabled
|       |-- *.load
|       `-- *.conf
|-- conf-enabled
|       `-- *.conf
|-- sites-enabled
|       `-- *.conf
          </pre>
          <ul>
                        <li>
                           <tt>apache2.conf</tt> is the main configuration
                           file. It puts the pieces together by including all remaining configuration
                           files when starting up the web server.
                        </li>

                        <li>
                           <tt>ports.conf</tt> is always included from the
                           main configuration file. It is used to determine the listening ports for
                           incoming connections, and this file can be customized anytime.
                        </li>

                        <li>
                           Configuration files in the <tt>mods-enabled/</tt>,
                           <tt>conf-enabled/</tt> and <tt>sites-enabled/</tt> directories contain
                           particular configuration snippets which manage modules, global configuration
                           fragments, or virtual host configurations, respectively.
                        </li>

                        <li>
                           They are activated by symlinking available
                           configuration files from their respective
                           *-available/ counterparts. These should be managed
                           by using our helpers
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enmod">a2enmod</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dismod">a2dismod</a>,
                           </tt>
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2ensite">a2ensite</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dissite">a2dissite</a>,
                            </tt>
                                and
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enconf">a2enconf</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2disconf">a2disconf</a>
                           </tt>. See their respective man pages for detailed information.
                        </li>

                        <li>
                           The binary is called apache2. Due to the use of
                           environment variables, in the default configuration, apache2 needs to be
                           started/stopped with <tt>/etc/init.d/apache2</tt> or <tt>apache2ctl</tt>.
                           <b>Calling <tt>/usr/bin/apache2</tt> directly will not work</b> with the
                           default configuration.
                        </li>
          </ul>
        </div>

        <div class="section_header">
            <div id="docroot"></div>
                Document Roots
        </div>

        <div class="content_section_text">
            <p>
                By default, Ubuntu does not allow access through the web browser to
                <em>any</em> file apart of those located in <tt>/var/www</tt>,
                <a href="http://httpd.apache.org/docs/2.4/mod/mod_userdir.html">public_html</a>
                directories (when enabled) and <tt>/usr/share</tt> (for web
                applications). If your site is using a web document root
                located elsewhere (such as in <tt>/srv</tt>) you may need to whitelist your
                document root directory in <tt>/etc/apache2/apache2.conf</tt>.
            </p>
            <p>
                The default Ubuntu document root is <tt>/var/www/html</tt>. You
                can make your own virtual hosts under /var/www. This is different
                to previous releases which provides better security out of the box.
            </p>
        </div>

        <div class="section_header">
          <div id="bugs"></div>
                Reporting Problems
        </div>
        <div class="content_section_text">
          <p>
                Please use the <tt>ubuntu-bug</tt> tool to report bugs in the
                Apache2 package with Ubuntu. However, check <a
                href="https://bugs.launchpad.net/ubuntu/+source/apache2">existing
                bug reports</a> before reporting a new bug.
          </p>
          <p>
                Please report bugs specific to modules (such as PHP and others)
                to respective packages, not to the web server itself.
          </p>
        </div>




      </div>
    </div>
    <div class="validator">
    <p>
      <a href="http://validator.w3.org/check?uri=referer"><img src="http://www.w3.org/Icons/valid-xhtml10" alt="Valid XHTML 1.0 Transitional" height="31" width="88" /></a>
    </p>
    </div>
  </body>
</html>

====================

GET /index.html HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Vary: Accept-Encoding
Content-Type: text/html


<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <!--
    Modified from the Debian original for Ubuntu
    Last updated: 2014-03-19
    See: https://launchpad.net/bugs/1288690
  -->
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <title>Apache2 Ubuntu Default Page: It works</title>
    <style type="text/css" media="screen">
  * {
    margin: 0px 0px 0px 0px;
    padding: 0px 0px 0px 0px;
  }

  body, html {
    padding: 3px 3px 3px 3px;

    background-color: #D8DBE2;

    font-family: Verdana, sans-serif;
    font-size: 11pt;
    text-align: center;
  }

  div.main_page {
    position: relative;
    display: table;

    width: 800px;

    margin-bottom: 3px;
    margin-left: auto;
    margin-right: auto;
    padding: 0px 0px 0px 0px;

    border-width: 2px;
    border-color: #212738;
    border-style: solid;

    background-color: #FFFFFF;

    text-align: center;
  }

  div.page_header {
    height: 99px;
    width: 100%;

    background-color: #F5F6F7;
  }

  div.page_header span {
    margin: 15px 0px 0px 50px;

    font-size: 180%;
    font-weight: bold;
  }

  div.page_header img {
    margin: 3px 0px 0px 40px;

    border: 0px 0px 0px;
  }

  div.table_of_contents {
    clear: left;

    min-width: 200px;

    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.table_of_contents_item {
    clear: left;

    width: 100%;

    margin: 4px 0px 0px 0px;

    background-color: #FFFFFF;

    color: #000000;
    text-align: left;
  }

  div.table_of_contents_item a {
    margin: 6px 0px 0px 6px;
  }

  div.content_section {
    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.content_section_text {
    padding: 4px 8px 4px 8px;

    color: #000000;
    font-size: 100%;
  }

  div.content_section_text pre {
    margin: 8px 0px 8px 0px;
    padding: 8px 8px 8px 8px;

    border-width: 1px;
    border-style: dotted;
    border-color: #000000;

    background-color: #F5F6F7;

    font-style: italic;
  }

  div.content_section_text p {
    margin-bottom: 6px;
  }

  div.content_section_text ul, div.content_section_text li {
    padding: 4px 8px 4px 16px;
  }

  div.section_header {
    padding: 3px 6px 3px 6px;

    background-color: #8E9CB2;

    color: #FFFFFF;
    font-weight: bold;
    font-size: 112%;
    text-align: center;
  }

  div.section_header_red {
    background-color: #CD214F;
  }

  div.section_header_grey {
    background-color: #9F9386;
  }

  .floating_element {
    position: relative;
    float: left;
  }

  div.table_of_contents_item a,
  div.content_section_text a {
    text-decoration: none;
    font-weight: bold;
  }

  div.table_of_contents_item a:link,
  div.table_of_contents_item a:visited,
  div.table_of_contents_item a:active {
    color: #000000;
  }

  div.table_of_contents_item a:hover {
    background-color: #000000;

    color: #FFFFFF;
  }

  div.content_section_text a:link,
  div.content_section_text a:visited,
   div.content_section_text a:active {
    background-color: #DCDFE6;

    color: #000000;
  }

  div.content_section_text a:hover {
    background-color: #000000;

    color: #DCDFE6;
  }

  div.validator {
  }
    </style>
  </head>
  <body>
    <div class="main_page">
      <div class="page_header floating_element">
        <img src="/icons/ubuntu-logo.png" alt="Ubuntu Logo" class="floating_element"/>
        <span class="floating_element">
          Apache2 Ubuntu Default Page
        </span>
      </div>
<!--      <div class="table_of_contents floating_element">
        <div class="section_header section_header_grey">
          TABLE OF CONTENTS
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#about">About</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#changes">Changes</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#scope">Scope</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#files">Config files</a>
        </div>
      </div>
-->
      <div class="content_section floating_element">


        <div class="section_header section_header_red">
          <div id="about"></div>
          It works!
        </div>
        <div class="content_section_text">
          <p>
                This is the default welcome page used to test the correct 
                operation of the Apache2 server after installation on Ubuntu systems.
                It is based on the equivalent page on Debian, from which the Ubuntu Apache
                packaging is derived.
                If you can read this page, it means that the Apache HTTP server installed at
                this site is working properly. You should <b>replace this file</b> (located at
                <tt>/var/www/html/index.html</tt>) before continuing to operate your HTTP server.
          </p>


          <p>
                If you are a normal user of this web site and don't know what this page is
                about, this probably means that the site is currently unavailable due to
                maintenance.
                If the problem persists, please contact the site's administrator.
          </p>

        </div>
        <div class="section_header">
          <div id="changes"></div>
                Configuration Overview
        </div>
        <div class="content_section_text">
          <p>
                Ubuntu's Apache2 default configuration is different from the
                upstream default configuration, and split into several files optimized for
                interaction with Ubuntu tools. The configuration system is
                <b>fully documented in
                /usr/share/doc/apache2/README.Debian.gz</b>. Refer to this for the full
                documentation. Documentation for the web server itself can be
                found by accessing the <a href="/manual">manual</a> if the <tt>apache2-doc</tt>
                package was installed on this server.

          </p>
          <p>
                The configuration layout for an Apache2 web server installation on Ubuntu systems is as follows:
          </p>
          <pre>
/etc/apache2/
|-- apache2.conf
|       `--  ports.conf
|-- mods-enabled
|       |-- *.load
|       `-- *.conf
|-- conf-enabled
|       `-- *.conf
|-- sites-enabled
|       `-- *.conf
          </pre>
          <ul>
                        <li>
                           <tt>apache2.conf</tt> is the main configuration
                           file. It puts the pieces together by including all remaining configuration
                           files when starting up the web server.
                        </li>

                        <li>
                           <tt>ports.conf</tt> is always included from the
                           main configuration file. It is used to determine the listening ports for
                           incoming connections, and this file can be customized anytime.
                        </li>

                        <li>
                           Configuration files in the <tt>mods-enabled/</tt>,
                           <tt>conf-enabled/</tt> and <tt>sites-enabled/</tt> directories contain
                           particular configuration snippets which manage modules, global configuration
                           fragments, or virtual host configurations, respectively.
                        </li>

                        <li>
                           They are activated by symlinking available
                           configuration files from their respective
                           *-available/ counterparts. These should be managed
                           by using our helpers
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enmod">a2enmod</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dismod">a2dismod</a>,
                           </tt>
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2ensite">a2ensite</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dissite">a2dissite</a>,
                            </tt>
                                and
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enconf">a2enconf</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2disconf">a2disconf</a>
                           </tt>. See their respective man pages for detailed information.
                        </li>

                        <li>
                           The binary is called apache2. Due to the use of
                           environment variables, in the default configuration, apache2 needs to be
                           started/stopped with <tt>/etc/init.d/apache2</tt> or <tt>apache2ctl</tt>.
                           <b>Calling <tt>/usr/bin/apache2</tt> directly will not work</b> with the
                           default configuration.
                        </li>
          </ul>
        </div>

        <div class="section_header">
            <div id="docroot"></div>
                Document Roots
        </div>

        <div class="content_section_text">
            <p>
                By default, Ubuntu does not allow access through the web browser to
                <em>any</em> file apart of those located in <tt>/var/www</tt>,
                <a href="http://httpd.apache.org/docs/2.4/mod/mod_userdir.html">public_html</a>
                directories (when enabled) and <tt>/usr/share</tt> (for web
                applications). If your site is using a web document root
                located elsewhere (such as in <tt>/srv</tt>) you may need to whitelist your
                document root directory in <tt>/etc/apache2/apache2.conf</tt>.
            </p>
            <p>
                The default Ubuntu document root is <tt>/var/www/html</tt>. You
                can make your own virtual hosts under /var/www. This is different
                to previous releases which provides better security out of the box.
            </p>
        </div>

        <div class="section_header">
          <div id="bugs"></div>
                Reporting Problems
        </div>
        <div class="content_section_text">
          <p>
                Please use the <tt>ubuntu-bug</tt> tool to report bugs in the
                Apache2 package with Ubuntu. However, check <a
                href="https://bugs.launchpad.net/ubuntu/+source/apache2">existing
                bug reports</a> before reporting a new bug.
          </p>
          <p>
                Please report bugs specific to modules (such as PHP and others)
                to respective packages, not to the web server itself.
          </p>
        </div>




      </div>
    </div>
    <div class="validator">
    <p>
      <a href="http://validator.w3.org/check?uri=referer"><img src="http://www.w3.org/Icons/valid-xhtml10" alt="Valid XHTML 1.0 Transitional" height="31" width="88" /></a>
    </p>
    </div>
  </body>
</html>

====================

GET /icons/ubuntu-logo.png HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Content-Type: image/png

�PNG

   IHDR   w   c   �~�  IDATx�{�S���bE��$�b}?Z�V�V+>�$#�ZD��@TD�PT(*�TX�(* (-�*�Q��$�{�����<����.�:�7���������s���������@DIǞ��Sr]�a�.Ǥ<���<�e����z\��yn��n�3�mt��vY2�C�<�7�v.ę�� $IA���w~�'};tR�Ɖ�N:bL�ŀJ��1?���%�ab�?EÏE�WH&C;w�9�n���1������z���@S�۞���kW%n�q�����(��u�oW�F�\��ZnP@�ϲ�7Q�J�.�~4d+�D˩;�:~�ĵ /C0�͎�h�^EE� �gF.�v|�K�~������XQ�G���H3
��P���ӿg���X���č f�S�!��ῦ��Jҳ��+"_�i�V�ٳ�� ��q��a:\Լ?�-����6s����:x�i�q ���پ�M]�}�J��d���a|����өl�=T�ִู󺠀�[?��ͧ��&S���w����M���pV6�~�;�Y� %�F�f�ݩx� ��d�Md�XT�[��T�^���|��m~��ulʋ��a?;Zkٽw^Q�ɪ�x��4j��AzV1g����&(qݶ�����U��"�#i���ߝIV�黯 no�ʃ]6Ϛ�~�{vRY�}��V�����RV\o��ϡ�?�Jj)�!��Y���m5d՚�F���昈���Bʊ���zi���k�}i|ۼ��
~V��x�1���o\C�f>D�^��/�,~��G���Z���!��֊b�v����=7�����`���Qٴ�T4�tT���*�=I�/����h{�T��]�"��)(��Y)g��^�jo����r9�ݚ�^����ot�&e��M9q��%�XUg�}��l�ZBU̞@��dd~��#H	̩:)#.��HM��_j��a'Y�������W�ZV@FV��cr�A���˟�2�԰a%�Qщ=��/�~�_W�K/��nJq���j�n?78��Y���G��\���>�ce��9�''��Z���b���%
&.�����8<�����̤77�{�L�4~�9�Y��1�?z��M6X��{�^\-�GF�NJ��$�wW0 P�pf��%��IF�>6�Q�O:I��^�7�����:.K�@���jA6�ZK�/�1^���֯紽���±p��ó���؉�?�`�'�������3�(2�����[v;&꾵��`�a���Ҍ��1�À�b?6��G���ﾜ|Y�˄ w$���tL�{xNBg��rͻ���;R�%����H�X�4v���+�=J�okݚ�T���`P�fūT���I����I/.��{xtu���-�y���T8���XQ��o�g��|C�z�eI/��m�ƨ��e�������:^�4}��B���FL�*F�",_�7u'���qV�IL�SC�.����'3(^�[Գҧ��DXvw�2L�p_��L�Kzqy�Lc����l���1��U=c�b��!۝�rګ�6Nɤ����b���p���Gz�	��5y8J����]���3��rҳ��|�^����4jܾ��>g�����<&%���fI���=EFƮBL��:���hڹI�%)��N���8�[����5~���=C��S���<*��vGj�P�lI��&�p�=@����sč`uk���������T�������<7�,S��(*y2˰��r�R�^'1�C��o\W��S.�ћ��o��]��q~2>�`��D���j��_�,�_��w)��W47r�_ߐ��q|�SN\��e��_�:8����A�������m�k�y��K�����h)).�E�aJ~�⠪���lEv�H2�ɏ%㄀�����S�C�ٔ��h���n�C]�����`����ȪU�����^�W��|�2��o80���M�h�¨��vIp��b<�r.2�|�|��m뭋�ޭ��Yo�z#�a┝�5��7�x��g5��R
cֺh�c�:6!4�"yQ��d_�yT��tD���T��y�	/�x/���N��ſ�>2AH@W�z�����:���E��/���]3�Q������Tq�m��Z٬[3K�C�=/�7�p�˾eĆu�j˞a�Wu<��d����o�l޽�w�S��g3[4��e���a(��%������72�	�D���=%���X�`�h����ĕo�H>�8�¶�Q���J\��I����]���|����31�=@��C���Z����J�Ө8�K�\��QD����k�D�y1�)��*q�������n��!h�(h l�x�$��W��1��{���T��1Vޘ�ߣ��s `6�����W�+=]O�ri���9��
%�W��U(qa��y���D\���SP�W��<�v�(q��
%��������Ma��p�7yĝ���P�*q��J\%.[��LT�(�q�����L�pY�E�N��ݥ�׸���
�	���`�69. GK�{=�m��B]��yG/M�ӁBX!�V�c��:�*hd@�g��7���(J��	, ?� ���%��%q�S#�}G����g]!ԉ������(	q�q�^�G�{?�V�"�2K@��X>�A�N�F`7)�4Pz$��\:	e{��:�o�����\/��'G{Bu$������l�$�;�yh)�bwIq%�B,�Po`"���V�3���*q�
�f'���u��W��E��6����Pg�7�ʐzy	,�9B��"����R�tJPq�3&��$�����]�&��v=��+#nk��%X����y}��_'�����d�����8O����P�	�ܗ��>#�9߂��B��d��<��8��.ԛ�Sf]��/��'��ڂ�uȈ�X�E�x�����S�B�kX�;D��qo�,#���/�I�:!��:���$��N�̋V�ւ��gU�@u�/��#1Wܝ0�u��?N�tq���c���C�4���;Q|�"�o�6��j��
H�m��n��x�Dcy�k� A:	^� 	,��2�l�`�֥��y��`�Ok�z�����~p�P�B��B0���uƀ�POܓ���5�$��"4�I`���� ��r3cu�?%
l!�d��E���17��ƤN`���Yx8�e`W��V�M��Y�j1��G��`+�`��6���u灣�<�Lֺ���{P��>к����jA�>ڤp>آ�o!�ֱP�ym�z>8���2=�,3K,�� ��D�6�*��
����'a꟬k3��*�ߥ�b��\0Fj�_UV�j��    IEND�B`�====================

GET //cgi-bin/php HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin/php was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET //cgi-bin/php5 HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin/php5 was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET //cgi-bin/php-cgi HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin/php-cgi was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET //cgi-bin/php.cgi HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin/php.cgi was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET //cgi-bin/php4 HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin/php4 was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET ///cgi-bin/php4 HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin/php4 was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET //cgi-bin//php4 HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /cgi-bin//php4 was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET /not-existed HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /not-existed was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

GET /end-with-slash/ HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /end-with-slash/ was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

HEAD / HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Vary: Accept-Encoding
Content-Type: text/html

====================

HEAD /index.html HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Vary: Accept-Encoding
Content-Type: text/html

====================

HEAD /icons/ubuntu-logo.png HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Content-Type: image/png

====================

HEAD /not-existed HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Type: text/html; charset=iso-8859-1

====================

POST / HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Vary: Accept-Encoding
Content-Type: text/html


<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <!--
    Modified from the Debian original for Ubuntu
    Last updated: 2014-03-19
    See: https://launchpad.net/bugs/1288690
  -->
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <title>Apache2 Ubuntu Default Page: It works</title>
    <style type="text/css" media="screen">
  * {
    margin: 0px 0px 0px 0px;
    padding: 0px 0px 0px 0px;
  }

  body, html {
    padding: 3px 3px 3px 3px;

    background-color: #D8DBE2;

    font-family: Verdana, sans-serif;
    font-size: 11pt;
    text-align: center;
  }

  div.main_page {
    position: relative;
    display: table;

    width: 800px;

    margin-bottom: 3px;
    margin-left: auto;
    margin-right: auto;
    padding: 0px 0px 0px 0px;

    border-width: 2px;
    border-color: #212738;
    border-style: solid;

    background-color: #FFFFFF;

    text-align: center;
  }

  div.page_header {
    height: 99px;
    width: 100%;

    background-color: #F5F6F7;
  }

  div.page_header span {
    margin: 15px 0px 0px 50px;

    font-size: 180%;
    font-weight: bold;
  }

  div.page_header img {
    margin: 3px 0px 0px 40px;

    border: 0px 0px 0px;
  }

  div.table_of_contents {
    clear: left;

    min-width: 200px;

    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.table_of_contents_item {
    clear: left;

    width: 100%;

    margin: 4px 0px 0px 0px;

    background-color: #FFFFFF;

    color: #000000;
    text-align: left;
  }

  div.table_of_contents_item a {
    margin: 6px 0px 0px 6px;
  }

  div.content_section {
    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.content_section_text {
    padding: 4px 8px 4px 8px;

    color: #000000;
    font-size: 100%;
  }

  div.content_section_text pre {
    margin: 8px 0px 8px 0px;
    padding: 8px 8px 8px 8px;

    border-width: 1px;
    border-style: dotted;
    border-color: #000000;

    background-color: #F5F6F7;

    font-style: italic;
  }

  div.content_section_text p {
    margin-bottom: 6px;
  }

  div.content_section_text ul, div.content_section_text li {
    padding: 4px 8px 4px 16px;
  }

  div.section_header {
    padding: 3px 6px 3px 6px;

    background-color: #8E9CB2;

    color: #FFFFFF;
    font-weight: bold;
    font-size: 112%;
    text-align: center;
  }

  div.section_header_red {
    background-color: #CD214F;
  }

  div.section_header_grey {
    background-color: #9F9386;
  }

  .floating_element {
    position: relative;
    float: left;
  }

  div.table_of_contents_item a,
  div.content_section_text a {
    text-decoration: none;
    font-weight: bold;
  }

  div.table_of_contents_item a:link,
  div.table_of_contents_item a:visited,
  div.table_of_contents_item a:active {
    color: #000000;
  }

  div.table_of_contents_item a:hover {
    background-color: #000000;

    color: #FFFFFF;
  }

  div.content_section_text a:link,
  div.content_section_text a:visited,
   div.content_section_text a:active {
    background-color: #DCDFE6;

    color: #000000;
  }

  div.content_section_text a:hover {
    background-color: #000000;

    color: #DCDFE6;
  }

  div.validator {
  }
    </style>
  </head>
  <body>
    <div class="main_page">
      <div class="page_header floating_element">
        <img src="/icons/ubuntu-logo.png" alt="Ubuntu Logo" class="floating_element"/>
        <span class="floating_element">
          Apache2 Ubuntu Default Page
        </span>
      </div>
<!--      <div class="table_of_contents floating_element">
        <div class="section_header section_header_grey">
          TABLE OF CONTENTS
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#about">About</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#changes">Changes</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#scope">Scope</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#files">Config files</a>
        </div>
      </div>
-->
      <div class="content_section floating_element">


        <div class="section_header section_header_red">
          <div id="about"></div>
          It works!
        </div>
        <div class="content_section_text">
          <p>
                This is the default welcome page used to test the correct 
                operation of the Apache2 server after installation on Ubuntu systems.
                It is based on the equivalent page on Debian, from which the Ubuntu Apache
                packaging is derived.
                If you can read this page, it means that the Apache HTTP server installed at
                this site is working properly. You should <b>replace this file</b> (located at
                <tt>/var/www/html/index.html</tt>) before continuing to operate your HTTP server.
          </p>


          <p>
                If you are a normal user of this web site and don't know what this page is
                about, this probably means that the site is currently unavailable due to
                maintenance.
                If the problem persists, please contact the site's administrator.
          </p>

        </div>
        <div class="section_header">
          <div id="changes"></div>
                Configuration Overview
        </div>
        <div class="content_section_text">
          <p>
                Ubuntu's Apache2 default configuration is different from the
                upstream default configuration, and split into several files optimized for
                interaction with Ubuntu tools. The configuration system is
                <b>fully documented in
                /usr/share/doc/apache2/README.Debian.gz</b>. Refer to this for the full
                documentation. Documentation for the web server itself can be
                found by accessing the <a href="/manual">manual</a> if the <tt>apache2-doc</tt>
                package was installed on this server.

          </p>
          <p>
                The configuration layout for an Apache2 web server installation on Ubuntu systems is as follows:
          </p>
          <pre>
/etc/apache2/
|-- apache2.conf
|       `--  ports.conf
|-- mods-enabled
|       |-- *.load
|       `-- *.conf
|-- conf-enabled
|       `-- *.conf
|-- sites-enabled
|       `-- *.conf
          </pre>
          <ul>
                        <li>
                           <tt>apache2.conf</tt> is the main configuration
                           file. It puts the pieces together by including all remaining configuration
                           files when starting up the web server.
                        </li>

                        <li>
                           <tt>ports.conf</tt> is always included from the
                           main configuration file. It is used to determine the listening ports for
                           incoming connections, and this file can be customized anytime.
                        </li>

                        <li>
                           Configuration files in the <tt>mods-enabled/</tt>,
                           <tt>conf-enabled/</tt> and <tt>sites-enabled/</tt> directories contain
                           particular configuration snippets which manage modules, global configuration
                           fragments, or virtual host configurations, respectively.
                        </li>

                        <li>
                           They are activated by symlinking available
                           configuration files from their respective
                           *-available/ counterparts. These should be managed
                           by using our helpers
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enmod">a2enmod</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dismod">a2dismod</a>,
                           </tt>
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2ensite">a2ensite</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dissite">a2dissite</a>,
                            </tt>
                                and
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enconf">a2enconf</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2disconf">a2disconf</a>
                           </tt>. See their respective man pages for detailed information.
                        </li>

                        <li>
                           The binary is called apache2. Due to the use of
                           environment variables, in the default configuration, apache2 needs to be
                           started/stopped with <tt>/etc/init.d/apache2</tt> or <tt>apache2ctl</tt>.
                           <b>Calling <tt>/usr/bin/apache2</tt> directly will not work</b> with the
                           default configuration.
                        </li>
          </ul>
        </div>

        <div class="section_header">
            <div id="docroot"></div>
                Document Roots
        </div>

        <div class="content_section_text">
            <p>
                By default, Ubuntu does not allow access through the web browser to
                <em>any</em> file apart of those located in <tt>/var/www</tt>,
                <a href="http://httpd.apache.org/docs/2.4/mod/mod_userdir.html">public_html</a>
                directories (when enabled) and <tt>/usr/share</tt> (for web
                applications). If your site is using a web document root
                located elsewhere (such as in <tt>/srv</tt>) you may need to whitelist your
                document root directory in <tt>/etc/apache2/apache2.conf</tt>.
            </p>
            <p>
                The default Ubuntu document root is <tt>/var/www/html</tt>. You
                can make your own virtual hosts under /var/www. This is different
                to previous releases which provides better security out of the box.
            </p>
        </div>

        <div class="section_header">
          <div id="bugs"></div>
                Reporting Problems
        </div>
        <div class="content_section_text">
          <p>
                Please use the <tt>ubuntu-bug</tt> tool to report bugs in the
                Apache2 package with Ubuntu. However, check <a
                href="https://bugs.launchpad.net/ubuntu/+source/apache2">existing
                bug reports</a> before reporting a new bug.
          </p>
          <p>
                Please report bugs specific to modules (such as PHP and others)
                to respective packages, not to the web server itself.
          </p>
        </div>




      </div>
    </div>
    <div class="validator">
    <p>
      <a href="http://validator.w3.org/check?uri=referer"><img src="http://www.w3.org/Icons/valid-xhtml10" alt="Valid XHTML 1.0 Transitional" height="31" width="88" /></a>
    </p>
    </div>
  </body>
</html>

====================

POST /index.html HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Vary: Accept-Encoding
Content-Type: text/html


<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <!--
    Modified from the Debian original for Ubuntu
    Last updated: 2014-03-19
    See: https://launchpad.net/bugs/1288690
  -->
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <title>Apache2 Ubuntu Default Page: It works</title>
    <style type="text/css" media="screen">
  * {
    margin: 0px 0px 0px 0px;
    padding: 0px 0px 0px 0px;
  }

  body, html {
    padding: 3px 3px 3px 3px;

    background-color: #D8DBE2;

    font-family: Verdana, sans-serif;
    font-size: 11pt;
    text-align: center;
  }

  div.main_page {
    position: relative;
    display: table;

    width: 800px;

    margin-bottom: 3px;
    margin-left: auto;
    margin-right: auto;
    padding: 0px 0px 0px 0px;

    border-width: 2px;
    border-color: #212738;
    border-style: solid;

    background-color: #FFFFFF;

    text-align: center;
  }

  div.page_header {
    height: 99px;
    width: 100%;

    background-color: #F5F6F7;
  }

  div.page_header span {
    margin: 15px 0px 0px 50px;

    font-size: 180%;
    font-weight: bold;
  }

  div.page_header img {
    margin: 3px 0px 0px 40px;

    border: 0px 0px 0px;
  }

  div.table_of_contents {
    clear: left;

    min-width: 200px;

    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.table_of_contents_item {
    clear: left;

    width: 100%;

    margin: 4px 0px 0px 0px;

    background-color: #FFFFFF;

    color: #000000;
    text-align: left;
  }

  div.table_of_contents_item a {
    margin: 6px 0px 0px 6px;
  }

  div.content_section {
    margin: 3px 3px 3px 3px;

    background-color: #FFFFFF;

    text-align: left;
  }

  div.content_section_text {
    padding: 4px 8px 4px 8px;

    color: #000000;
    font-size: 100%;
  }

  div.content_section_text pre {
    margin: 8px 0px 8px 0px;
    padding: 8px 8px 8px 8px;

    border-width: 1px;
    border-style: dotted;
    border-color: #000000;

    background-color: #F5F6F7;

    font-style: italic;
  }

  div.content_section_text p {
    margin-bottom: 6px;
  }

  div.content_section_text ul, div.content_section_text li {
    padding: 4px 8px 4px 16px;
  }

  div.section_header {
    padding: 3px 6px 3px 6px;

    background-color: #8E9CB2;

    color: #FFFFFF;
    font-weight: bold;
    font-size: 112%;
    text-align: center;
  }

  div.section_header_red {
    background-color: #CD214F;
  }

  div.section_header_grey {
    background-color: #9F9386;
  }

  .floating_element {
    position: relative;
    float: left;
  }

  div.table_of_contents_item a,
  div.content_section_text a {
    text-decoration: none;
    font-weight: bold;
  }

  div.table_of_contents_item a:link,
  div.table_of_contents_item a:visited,
  div.table_of_contents_item a:active {
    color: #000000;
  }

  div.table_of_contents_item a:hover {
    background-color: #000000;

    color: #FFFFFF;
  }

  div.content_section_text a:link,
  div.content_section_text a:visited,
   div.content_section_text a:active {
    background-color: #DCDFE6;

    color: #000000;
  }

  div.content_section_text a:hover {
    background-color: #000000;

    color: #DCDFE6;
  }

  div.validator {
  }
    </style>
  </head>
  <body>
    <div class="main_page">
      <div class="page_header floating_element">
        <img src="/icons/ubuntu-logo.png" alt="Ubuntu Logo" class="floating_element"/>
        <span class="floating_element">
          Apache2 Ubuntu Default Page
        </span>
      </div>
<!--      <div class="table_of_contents floating_element">
        <div class="section_header section_header_grey">
          TABLE OF CONTENTS
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#about">About</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#changes">Changes</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#scope">Scope</a>
        </div>
        <div class="table_of_contents_item floating_element">
          <a href="#files">Config files</a>
        </div>
      </div>
-->
      <div class="content_section floating_element">


        <div class="section_header section_header_red">
          <div id="about"></div>
          It works!
        </div>
        <div class="content_section_text">
          <p>
                This is the default welcome page used to test the correct 
                operation of the Apache2 server after installation on Ubuntu systems.
                It is based on the equivalent page on Debian, from which the Ubuntu Apache
                packaging is derived.
                If you can read this page, it means that the Apache HTTP server installed at
                this site is working properly. You should <b>replace this file</b> (located at
                <tt>/var/www/html/index.html</tt>) before continuing to operate your HTTP server.
          </p>


          <p>
                If you are a normal user of this web site and don't know what this page is
                about, this probably means that the site is currently unavailable due to
                maintenance.
                If the problem persists, please contact the site's administrator.
          </p>

        </div>
        <div class="section_header">
          <div id="changes"></div>
                Configuration Overview
        </div>
        <div class="content_section_text">
          <p>
                Ubuntu's Apache2 default configuration is different from the
                upstream default configuration, and split into several files optimized for
                interaction with Ubuntu tools. The configuration system is
                <b>fully documented in
                /usr/share/doc/apache2/README.Debian.gz</b>. Refer to this for the full
                documentation. Documentation for the web server itself can be
                found by accessing the <a href="/manual">manual</a> if the <tt>apache2-doc</tt>
                package was installed on this server.

          </p>
          <p>
                The configuration layout for an Apache2 web server installation on Ubuntu systems is as follows:
          </p>
          <pre>
/etc/apache2/
|-- apache2.conf
|       `--  ports.conf
|-- mods-enabled
|       |-- *.load
|       `-- *.conf
|-- conf-enabled
|       `-- *.conf
|-- sites-enabled
|       `-- *.conf
          </pre>
          <ul>
                        <li>
                           <tt>apache2.conf</tt> is the main configuration
                           file. It puts the pieces together by including all remaining configuration
                           files when starting up the web server.
                        </li>

                        <li>
                           <tt>ports.conf</tt> is always included from the
                           main configuration file. It is used to determine the listening ports for
                           incoming connections, and this file can be customized anytime.
                        </li>

                        <li>
                           Configuration files in the <tt>mods-enabled/</tt>,
                           <tt>conf-enabled/</tt> and <tt>sites-enabled/</tt> directories contain
                           particular configuration snippets which manage modules, global configuration
                           fragments, or virtual host configurations, respectively.
                        </li>

                        <li>
                           They are activated by symlinking available
                           configuration files from their respective
                           *-available/ counterparts. These should be managed
                           by using our helpers
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enmod">a2enmod</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dismod">a2dismod</a>,
                           </tt>
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2ensite">a2ensite</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2dissite">a2dissite</a>,
                            </tt>
                                and
                           <tt>
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2enconf">a2enconf</a>,
                                <a href="http://manpages.debian.org/cgi-bin/man.cgi?query=a2disconf">a2disconf</a>
                           </tt>. See their respective man pages for detailed information.
                        </li>

                        <li>
                           The binary is called apache2. Due to the use of
                           environment variables, in the default configuration, apache2 needs to be
                           started/stopped with <tt>/etc/init.d/apache2</tt> or <tt>apache2ctl</tt>.
                           <b>Calling <tt>/usr/bin/apache2</tt> directly will not work</b> with the
                           default configuration.
                        </li>
          </ul>
        </div>

        <div class="section_header">
            <div id="docroot"></div>
                Document Roots
        </div>

        <div class="content_section_text">
            <p>
                By default, Ubuntu does not allow access through the web browser to
                <em>any</em> file apart of those located in <tt>/var/www</tt>,
                <a href="http://httpd.apache.org/docs/2.4/mod/mod_userdir.html">public_html</a>
                directories (when enabled) and <tt>/usr/share</tt> (for web
                applications). If your site is using a web document root
                located elsewhere (such as in <tt>/srv</tt>) you may need to whitelist your
                document root directory in <tt>/etc/apache2/apache2.conf</tt>.
            </p>
            <p>
                The default Ubuntu document root is <tt>/var/www/html</tt>. You
                can make your own virtual hosts under /var/www. This is different
                to previous releases which provides better security out of the box.
            </p>
        </div>

        <div class="section_header">
          <div id="bugs"></div>
                Reporting Problems
        </div>
        <div class="content_section_text">
          <p>
                Please use the <tt>ubuntu-bug</tt> tool to report bugs in the
                Apache2 package with Ubuntu. However, check <a
                href="https://bugs.launchpad.net/ubuntu/+source/apache2">existing
                bug reports</a> before reporting a new bug.
          </p>
          <p>
                Please report bugs specific to modules (such as PHP and others)
                to respective packages, not to the web server itself.
          </p>
        </div>




      </div>
    </div>
    <div class="validator">
    <p>
      <a href="http://validator.w3.org/check?uri=referer"><img src="http://www.w3.org/Icons/valid-xhtml10" alt="Valid XHTML 1.0 Transitional" height="31" width="88" /></a>
    </p>
    </div>
  </body>
</html>

====================

POST /icons/ubuntu-logo.png HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Last-Modified: <GMT>
ETag: "<ETAG>"
Accept-Ranges: bytes
Content-Length: <...>
Content-Type: image/png

�PNG

   IHDR   w   c   �~�  IDATx�{�S���bE��$�b}?Z�V�V+>�$#�ZD��@TD�PT(*�TX�(* (-�*�Q��$�{�����<����.�:�7���������s���������@DIǞ��Sr]�a�.Ǥ<���<�e����z\��yn��n�3�mt��vY2�C�<�7�v.ę�� $IA���w~�'};tR�Ɖ�N:bL�ŀJ��1?���%�ab�?EÏE�WH&C;w�9�n���1������z���@S�۞���kW%n�q�����(��u�oW�F�\��ZnP@�ϲ�7Q�J�.�~4d+�D˩;�:~�ĵ /C0�͎�h�^EE� �gF.�v|�K�~������XQ�G���H3
��P���ӿg���X���č f�S�!��ῦ��Jҳ��+"_�i�V�ٳ�� ��q��a:\Լ?�-����6s����:x�i�q ���پ�M]�}�J��d���a|����өl�=T�ִู󺠀�[?��ͧ��&S���w����M���pV6�~�;�Y� %�F�f�ݩx� ��d�Md�XT�[��T�^���|��m~��ulʋ��a?;Zkٽw^Q�ɪ�x��4j��AzV1g����&(qݶ�����U��"�#i���ߝIV�黯 no�ʃ]6Ϛ�~�{vRY�}��V�����RV\o��ϡ�?�Jj)�!��Y���m5d՚�F���昈���Bʊ���zi���k�}i|ۼ��
~V��x�1���o\C�f>D�^��/�,~��G���Z���!��֊b�v����=7�����`���Qٴ�T4�tT���*�=I�/����h{�T��]�"��)(��Y)g��^�jo����r9�ݚ�^����ot�&e��M9q��%�XUg�}��l�ZBU̞@��dd~��#H	̩:)#.��HM��_j��a'Y�������W�ZV@FV��cr�A���˟�2�԰a%�Qщ=��/�~�_W�K/��nJq���j�n?78��Y���G��\���>�ce��9�''��Z���b���%
&.�����8<�����̤77�{�L�4~�9�Y��1�?z��M6X��{�^\-�GF�NJ��$�wW0 P�pf��%��IF�>6�Q�O:I��^�7�����:.K�@���jA6�ZK�/�1^���֯紽���±p��ó���؉�?�`�'�������3�(2�����[v;&꾵��`�a���Ҍ��1�À�b?6��G���ﾜ|Y�˄ w$���tL�{xNBg��rͻ���;R�%����H�X�4v���+�=J�okݚ�T���`P�fūT���I����I/.��{xtu���-�y���T8���XQ��o�g��|C�z�eI/��m�ƨ��e�������:^�4}��B���FL�*F�",_�7u'���qV�IL�SC�.����'3(^�[Գҧ��DXvw�2L�p_��L�Kzqy�Lc����l���1��U=c�b��!۝�rګ�6Nɤ����b���p���Gz�	��5y8J����]���3��rҳ��|�^����4jܾ��>g�����<&%���fI���=EFƮBL��:���hڹI�%)��N���8�[����5~���=C��S���<*��vGj�P�lI��&�p�=@����sč`uk���������T�������<7�,S��(*y2˰��r�R�^'1�C��o\W��S.�ћ��o��]��q~2>�`��D���j��_�,�_��w)��W47r�_ߐ��q|�SN\��e��_�:8����A�������m�k�y��K�����h)).�E�aJ~�⠪���lEv�H2�ɏ%㄀�����S�C�ٔ��h���n�C]�����`����ȪU�����^�W��|�2��o80���M�h�¨��vIp��b<�r.2�|�|��m뭋�ޭ��Yo�z#�a┝�5��7�x��g5��R
cֺh�c�:6!4�"yQ��d_�yT��tD���T��y�	/�x/���N��ſ�>2AH@W�z�����:���E��/���]3�Q������Tq�m��Z٬[3K�C�=/�7�p�˾eĆu�j˞a�Wu<��d����o�l޽�w�S��g3[4��e���a(��%������72�	�D���=%���X�`�h����ĕo�H>�8�¶�Q���J\��I����]���|����31�=@��C���Z����J�Ө8�K�\��QD����k�D�y1�)��*q�������n��!h�(h l�x�$��W��1��{���T��1Vޘ�ߣ��s `6�����W�+=]O�ri���9��
%�W��U(qa��y���D\���SP�W��<�v�(q��
%��������Ma��p�7yĝ���P�*q��J\%.[��LT�(�q�����L�pY�E�N��ݥ�׸���
�	���`�69. GK�{=�m��B]��yG/M�ӁBX!�V�c��:�*hd@�g��7���(J��	, ?� ���%��%q�S#�}G����g]!ԉ������(	q�q�^�G�{?�V�"�2K@��X>�A�N�F`7)�4Pz$��\:	e{��:�o�����\/��'G{Bu$������l�$�;�yh)�bwIq%�B,�Po`"���V�3���*q�
�f'���u��W��E��6����Pg�7�ʐzy	,�9B��"����R�tJPq�3&��$�����]�&��v=��+#nk��%X����y}��_'�����d�����8O����P�	�ܗ��>#�9߂��B��d��<��8��.ԛ�Sf]��/��'��ڂ�uȈ�X�E�x�����S�B�kX�;D��qo�,#���/�I�:!��:���$��N�̋V�ւ��gU�@u�/��#1Wܝ0�u��?N�tq���c���C�4���;Q|�"�o�6��j��
H�m��n��x�Dcy�k� A:	^� 	,��2�l�`�֥��y��`�Ok�z�����~p�P�B��B0���uƀ�POܓ���5�$��"4�I`���� ��r3cu�?%
l!�d��E���17��ƤN`���Yx8�e`W��V�M��Y�j1��G��`+�`��6���u灣�<�Lֺ���{P��>к����jA�>ڤp>آ�o!�ֱP�ym�z>8���2=�,3K,�� ��D�6�*��
����'a꟬k3��*�ߥ�b��\0Fj�_UV�j��    IEND�B`�====================

POST /not-existed HTTP/1.1
--------------------
HTTP/1.1 404 Not Found
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>404 Not Found</title>
</head><body>
<h1>Not Found</h1>
<p>The requested URL /not-existed was not found on this server.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

OPTIONS / HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: text/html

====================

OPTIONS /index.html HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: text/html

====================

OPTIONS /icons/ubuntu-logo.png HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: image/png

====================

OPTIONS /not-existed HTTP/1.1
--------------------
HTTP/1.1 200 OK
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>

====================

PUT / HTTP/1.1
--------------------
HTTP/1.1 405 Method Not Allowed
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>405 Method Not Allowed</title>
</head><body>
<h1>Method Not Allowed</h1>
<p>The requested method PUT is not allowed for the URL /index.html.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

PUT /index.html HTTP/1.1
--------------------
HTTP/1.1 405 Method Not Allowed
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>405 Method Not Allowed</title>
</head><body>
<h1>Method Not Allowed</h1>
<p>The requested method PUT is not allowed for the URL /index.html.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

PUT /icons/ubuntu-logo.png HTTP/1.1
--------------------
HTTP/1.1 405 Method Not Allowed
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>405 Method Not Allowed</title>
</head><body>
<h1>Method Not Allowed</h1>
<p>The requested method PUT is not allowed for the URL /icons/ubuntu-logo.png.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

PUT /not-existed HTTP/1.1
--------------------
HTTP/1.1 405 Method Not Allowed
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>405 Method Not Allowed</title>
</head><body>
<h1>Method Not Allowed</h1>
<p>The requested method PUT is not allowed for the URL /not-existed.</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

CONNECT / HTTP/1.1
--------------------
HTTP/1.1 400 Bad Request
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>400 Bad Request</title>
</head><body>
<h1>Bad Request</h1>
<p>Your browser sent a request that this server could not understand.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

CONNECT /index.html HTTP/1.1
--------------------
HTTP/1.1 400 Bad Request
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>400 Bad Request</title>
</head><body>
<h1>Bad Request</h1>
<p>Your browser sent a request that this server could not understand.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

CONNECT /icons/ubuntu-logo.png HTTP/1.1
--------------------
HTTP/1.1 400 Bad Request
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>400 Bad Request</title>
</head><body>
<h1>Bad Request</h1>
<p>Your browser sent a request that this server could not understand.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

CONNECT /not-existed HTTP/1.1
--------------------
HTTP/1.1 400 Bad Request
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>400 Bad Request</title>
</head><body>
<h1>Bad Request</h1>
<p>Your browser sent a request that this server could not understand.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

INVALID / HTTP/1.1
--------------------
HTTP/1.1 501 Not Implemented
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>501 Not Implemented</title>
</head><body>
<h1>Not Implemented</h1>
<p>INVALID to /index.html not supported.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

INVALID /index.html HTTP/1.1
--------------------
HTTP/1.1 501 Not Implemented
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>501 Not Implemented</title>
</head><body>
<h1>Not Implemented</h1>
<p>INVALID to /index.html not supported.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

INVALID /not-existed HTTP/1.1
--------------------
HTTP/1.1 501 Not Implemented
Date: <GMT>
Server: Apache/2.4.7 (Ubuntu)
Allow: GET,HEAD,POST,OPTIONS
Content-Length: <...>
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>501 Not Implemented</title>
</head><body>
<h1>Not Implemented</h1>
<p>INVALID to /not-existed not supported.<br />
</p>
<hr>
<address>Apache/<VERSION> (Ubuntu) Server at <Host> Port <Port></address>
</body></html>
====================

