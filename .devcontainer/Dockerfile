FROM qmcgaw/godevcontainer
USER root
RUN apk add bash
USER vscode
# Git prompt
ENV USERNAME=vscode
RUN git clone https://github.com/magicmonty/bash-git-prompt.git /home/${USERNAME}/.bash-git-prompt --depth=1
RUN echo -e "\nif [ -f '/home/${USERNAME}/.bash-git-prompt/gitprompt.sh' ]; then\nGIT_PROMPT_ONLY_IN_REPO=1\nsource /home/${USERNAME}/.bash-git-prompt/gitprompt.sh\nfi" >> /home/${USERNAME}/.bashrc
